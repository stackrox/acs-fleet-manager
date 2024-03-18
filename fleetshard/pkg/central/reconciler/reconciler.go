// Package reconciler provides update, delete and create logic for managing Central instances.
package reconciler

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"sort"
	"sync/atomic"
	"time"

	"github.com/golang/glog"
	"github.com/hashicorp/go-multierror"
	openshiftRouteV1 "github.com/openshift/api/route/v1"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/fleetshard/config"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/charts"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/cloudprovider"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/postgres"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/cipher"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/k8s"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/util"
	centralConstants "github.com/stackrox/acs-fleet-manager/internal/dinosaur/constants"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
	"github.com/stackrox/acs-fleet-manager/pkg/features"
	centralNotifierUtils "github.com/stackrox/rox/central/notifiers/utils"
	"github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/random"
	"gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	corev1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/pointer"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	yaml2 "sigs.k8s.io/yaml"
)

// FreeStatus ...
const (
	FreeStatus int32 = iota
	BlockedStatus

	PauseReconcileAnnotation = "stackrox.io/pause-reconcile"

	helmReleaseName = "tenant-resources"

	centralPVCAnnotationKey   = "platform.stackrox.io/obsolete-central-pvc"
	managedServicesAnnotation = "platform.stackrox.io/managed-services"
	envAnnotationKey          = "rhacs.redhat.com/environment"
	clusterNameAnnotationKey  = "rhacs.redhat.com/cluster-name"
	orgNameAnnotationKey      = "rhacs.redhat.com/org-name"

	labelManagedByFleetshardValue = "rhacs-fleetshard"
	instanceLabelKey              = "app.kubernetes.io/instance"
	instanceTypeLabelKey          = "rhacs.redhat.com/instance-type"
	managedByLabelKey             = "app.kubernetes.io/managed-by"
	orgIDLabelKey                 = "rhacs.redhat.com/org-id"
	tenantIDLabelKey              = "rhacs.redhat.com/tenant"
	centralExpiredAtKey           = "rhacs.redhat.com/expired-at"

	auditLogNotifierKey  = "com.redhat.rhacs.auditLogNotifier"
	auditLogNotifierName = "Platform Audit Logs"
	auditLogTenantIDKey  = "tenant_id"

	dbUserTypeAnnotation = "platform.stackrox.io/user-type"
	dbUserTypeMaster     = "master"
	dbUserTypeCentral    = "central"
	dbCentralUserName    = "rhacs_central"

	centralDbSecretName        = "central-db-password" // pragma: allowlist secret
	centralDbOverrideConfigMap = "central-db-override"
	centralDeletePollInterval  = 5 * time.Second

	centralEncryptionKeySecretName = "central-encryption-key-chain" // pragma: allowlist secret

	sensibleDeclarativeConfigSecretName = "cloud-service-sensible-declarative-configs" // pragma: allowlist secret
	manualDeclarativeConfigSecretName   = "cloud-service-manual-declarative-configs"   // pragma: allowlist secret

	authProviderDeclarativeConfigKey = "default-sso-auth-provider"
	additionalAuthProviderConfigKey  = "additional-auth-provider"

	tenantImagePullSecretName = "stackrox" // pragma: allowlist secret
)

type verifyAuthProviderExistsFunc func(ctx context.Context, central private.ManagedCentral, client ctrlClient.Client) (bool, error)
type needsReconcileFunc func(changed bool, central *v1alpha1.Central, storedSecrets []string) bool
type restoreCentralSecretsFunc func(ctx context.Context, remoteCentral private.ManagedCentral) error
type areSecretsStoredFunc func(secretsStored []string) bool

// CentralReconcilerOptions are the static options for creating a reconciler.
type CentralReconcilerOptions struct {
	UseRoutes             bool
	WantsAuthProvider     bool
	EgressProxyImage      string
	ManagedDBEnabled      bool
	Telemetry             config.Telemetry
	ClusterName           string
	Environment           string
	AuditLogging          config.AuditLogging
	TenantImagePullSecret string
	RouteParameters       config.RouteConfig
}

// CentralReconciler is a reconciler tied to a one Central instance. It installs, updates and deletes Central instances
// in its Reconcile function.
type CentralReconciler struct {
	client                 ctrlClient.Client
	fleetmanagerClient     *fleetmanager.Client
	central                private.ManagedCentral
	status                 *int32
	lastCentralHash        [16]byte
	lastCentralHashTime    time.Time
	useRoutes              bool
	Resources              bool
	routeService           *k8s.RouteService
	secretBackup           *k8s.SecretBackup
	secretCipher           cipher.Cipher
	egressProxyImage       string
	telemetry              config.Telemetry
	clusterName            string
	environment            string
	auditLogging           config.AuditLogging
	encryptionKeyGenerator cipher.KeyGenerator

	managedDBEnabled            bool
	managedDBProvisioningClient cloudprovider.DBClient
	managedDBInitFunc           postgres.CentralDBInitFunc

	resourcesChart *chart.Chart

	wantsAuthProvider      bool
	hasAuthProvider        bool
	verifyAuthProviderFunc verifyAuthProviderExistsFunc
	tenantImagePullSecret  []byte
	clock                  clock

	areSecretsStoredFunc      areSecretsStoredFunc
	needsReconcileFunc        needsReconcileFunc
	restoreCentralSecretsFunc restoreCentralSecretsFunc
}

// Reconcile takes a private.ManagedCentral and tries to install it into the cluster managed by the fleet-shard.
// It tries to create a namespace for the Central and applies necessary updates to the resource.
// TODO(sbaumer): Check correct Central gets reconciled
// TODO(sbaumer): Should an initial ManagedCentral be added on reconciler creation?
func (r *CentralReconciler) Reconcile(ctx context.Context, remoteCentral private.ManagedCentral) (*private.DataPlaneCentralStatus, error) {
	// Only allow to start reconcile function once
	if !atomic.CompareAndSwapInt32(r.status, FreeStatus, BlockedStatus) {
		return nil, ErrBusy
	}
	defer atomic.StoreInt32(r.status, FreeStatus)

	centralHash, err := r.computeCentralHash(remoteCentral)
	if err != nil {
		return nil, errors.Wrap(err, "computing central hash")
	}

	central, err := r.getInstanceConfig(&remoteCentral)
	if err != nil {
		return nil, err
	}

	shouldUpdateCentralHash := false
	defer func() {
		if shouldUpdateCentralHash {
			r.lastCentralHash = centralHash
			r.lastCentralHashTime = time.Now()
		} else {
			r.lastCentralHash = [16]byte{}
		}
	}()

	changed := r.centralChanged(centralHash)

	needsReconcile := r.needsReconcileFunc(changed, central, remoteCentral.Metadata.SecretsStored)

	if !needsReconcile && r.shouldSkipReadyCentral(remoteCentral) {
		shouldUpdateCentralHash = true
		return nil, ErrCentralNotChanged
	}

	glog.Infof("Start reconcile central %s/%s", remoteCentral.Metadata.Namespace, remoteCentral.Metadata.Name)

	remoteCentralNamespace := remoteCentral.Metadata.Namespace

	if remoteCentral.Metadata.DeletionTimestamp != "" {
		status, err := r.reconcileInstanceDeletion(ctx, &remoteCentral, central)
		shouldUpdateCentralHash = err == nil
		return status, err
	}

	if err := r.reconcileNamespace(ctx, remoteCentral); err != nil {
		return nil, errors.Wrapf(err, "unable to ensure that namespace %s exists", remoteCentralNamespace)
	}

	if len(r.tenantImagePullSecret) > 0 {
		err = r.ensureImagePullSecretConfigured(ctx, remoteCentralNamespace, tenantImagePullSecretName, r.tenantImagePullSecret)
		if err != nil {
			return nil, err
		}
	}

	err = r.restoreCentralSecretsFunc(ctx, remoteCentral)
	if err != nil {
		return nil, err
	}

	err = r.ensureEncryptionKeySecretExists(ctx, remoteCentralNamespace)
	if err != nil {
		return nil, err
	}

	if err := r.ensureChartResourcesExist(ctx, remoteCentral); err != nil {
		return nil, errors.Wrapf(err, "unable to install chart resource for central %s/%s", central.GetNamespace(), central.GetName())
	}

	if err = r.reconcileCentralDBConfig(ctx, &remoteCentral, central); err != nil {
		return nil, err
	}

	if err = r.reconcileDeclarativeConfigurationData(ctx, remoteCentral); err != nil {
		return nil, err
	}

	if err := r.reconcileAdminPasswordGeneration(central); err != nil {
		return nil, err
	}

	if err = r.reconcileCentral(ctx, &remoteCentral, central); err != nil {
		return nil, err
	}

	centralTLSSecretFound := true // pragma: allowlist secret
	if r.useRoutes {
		if err := r.ensureRoutesExist(ctx, remoteCentral); err != nil {
			if errors.Is(err, k8s.ErrCentralTLSSecretNotFound) {
				centralTLSSecretFound = false // pragma: allowlist secret
			} else {
				return nil, errors.Wrap(err, "updating routes")
			}
		}
	}

	// Check whether deployment is ready.
	centralDeploymentReady, err := isCentralDeploymentReady(ctx, r.client, remoteCentral.Metadata.Namespace)
	if err != nil {
		return nil, err
	}

	if err = r.ensureSecretHasOwnerReference(ctx, k8s.CentralTLSSecretName, &remoteCentral, central); err != nil {
		return nil, err
	}

	if !centralDeploymentReady || !centralTLSSecretFound {
		if isRemoteCentralProvisioning(remoteCentral) && !needsReconcile { // no changes detected, wait until central become ready
			return nil, ErrCentralNotChanged
		}
		return installingStatus(), nil
	}

	exists, err := r.ensureAuthProviderExists(ctx, remoteCentral)
	if err != nil {
		return nil, err
	}
	if !exists {
		glog.Infof("Default auth provider for central %s/%s is not yet ready.",
			central.GetNamespace(), central.GetName())
		return nil, ErrCentralNotChanged
	}

	status, err := r.collectReconciliationStatus(ctx, &remoteCentral)
	if err != nil {
		return nil, err
	}

	shouldUpdateCentralHash = true

	logStatus := *status
	logStatus.Secrets = obscureSecrets(status.Secrets)
	glog.Infof("Returning central status %+v", logStatus)

	return status, nil
}

func (r *CentralReconciler) getInstanceConfig(remoteCentral *private.ManagedCentral) (*v1alpha1.Central, error) {
	var central = new(v1alpha1.Central)
	if err := yaml2.Unmarshal([]byte(remoteCentral.Spec.CentralCRYAML), central); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal central yaml")
	}
	if err := r.applyCentralConfig(remoteCentral, central); err != nil {
		return nil, err
	}
	return central, nil
}

func (r *CentralReconciler) applyCentralConfig(remoteCentral *private.ManagedCentral, central *v1alpha1.Central) error {
	r.applyTelemetry(remoteCentral, central)
	r.applyRoutes(central)
	r.applyProxyConfig(central)
	r.applyDeclarativeConfig(central)
	r.applyAnnotations(remoteCentral, central)
	return nil
}

func (r *CentralReconciler) applyAnnotations(remoteCentral *private.ManagedCentral, central *v1alpha1.Central) {
	if central.Spec.Customize == nil {
		central.Spec.Customize = &v1alpha1.CustomizeSpec{}
	}
	if central.Spec.Customize.Annotations == nil {
		central.Spec.Customize.Annotations = map[string]string{}
	}
	central.Spec.Customize.Annotations[envAnnotationKey] = r.environment
	central.Spec.Customize.Annotations[clusterNameAnnotationKey] = r.clusterName
	if remoteCentral.Metadata.ExpiredAt != nil {
		central.Spec.Customize.Annotations[centralExpiredAtKey] = remoteCentral.Metadata.ExpiredAt.Format(time.RFC3339)
	}
}

func (r *CentralReconciler) applyDeclarativeConfig(central *v1alpha1.Central) {
	if central.Spec.Central == nil {
		central.Spec.Central = &v1alpha1.CentralComponentSpec{}
	}
	declarativeConfig := &v1alpha1.DeclarativeConfiguration{
		Secrets: []v1alpha1.LocalSecretReference{
			{
				Name: sensibleDeclarativeConfigSecretName,
			},
			{
				Name: manualDeclarativeConfigSecretName,
			},
		},
	}

	central.Spec.Central.DeclarativeConfiguration = declarativeConfig
}

func (r *CentralReconciler) applyProxyConfig(central *v1alpha1.Central) {
	if central.Spec.Customize == nil {
		central.Spec.Customize = &v1alpha1.CustomizeSpec{}
	}
	auditLoggingURL := url.URL{Host: r.auditLogging.Endpoint(false)}
	kubernetesURL := url.URL{
		Host: "kubernetes.default.svc.cluster.local.:443",
	}
	envVars := getProxyEnvVars(central.Namespace, auditLoggingURL, kubernetesURL)
	central.Spec.Customize.EnvVars = append(central.Spec.Customize.EnvVars, envVars...)
}

func (r *CentralReconciler) applyRoutes(central *v1alpha1.Central) {
	if central.Spec.Central == nil {
		central.Spec.Central = &v1alpha1.CentralComponentSpec{}
	}
	exposure := &v1alpha1.Exposure{
		Route: &v1alpha1.ExposureRoute{
			Enabled: pointer.Bool(r.useRoutes),
		},
	}
	central.Spec.Central.Exposure = exposure
}

func (r *CentralReconciler) applyTelemetry(remoteCentral *private.ManagedCentral, central *v1alpha1.Central) {
	if central.Spec.Central == nil {
		central.Spec.Central = &v1alpha1.CentralComponentSpec{}
	}
	// Telemetry is always enabled, but the key is set to DISABLED for probe and other internal instances.
	// Cloud-service specificity: empty key also disables telemetry to prevent reporting to the self-managed bucket.
	key := r.telemetry.StorageKey
	if remoteCentral.Metadata.Internal || key == "" {
		key = "DISABLED"
	}
	telemetry := &v1alpha1.Telemetry{
		Enabled: pointer.Bool(true),
		Storage: &v1alpha1.TelemetryStorage{
			Endpoint: &r.telemetry.StorageEndpoint,
			Key:      &key,
		},
	}
	central.Spec.Central.Telemetry = telemetry
}

func (r *CentralReconciler) restoreCentralSecrets(ctx context.Context, remoteCentral private.ManagedCentral) error {
	restoreSecrets := []string{}
	for _, secretName := range remoteCentral.Metadata.SecretsStored { // pragma: allowlist secret
		exists, err := r.checkSecretExists(ctx, remoteCentral.Metadata.Namespace, secretName)
		if err != nil {
			return err
		}

		if !exists {
			restoreSecrets = append(restoreSecrets, secretName)
		}
	}

	if len(restoreSecrets) == 0 {
		// nothing to restore
		return nil
	}

	glog.Info(fmt.Sprintf("Restore secret for tenant: %s/%s", remoteCentral.Id, r.central.Metadata.Namespace), restoreSecrets)
	central, _, err := r.fleetmanagerClient.PrivateAPI().GetCentral(ctx, remoteCentral.Id)
	if err != nil {
		return fmt.Errorf("loading secrets for central %s: %w", remoteCentral.Id, err)
	}

	decryptedSecrets, err := r.decryptSecrets(central.Metadata.Secrets)
	if err != nil {
		return fmt.Errorf("decrypting secrets for central %s: %w", central.Id, err)
	}

	for _, secretName := range restoreSecrets { // pragma: allowlist secret
		secretToRestore, secretFound := decryptedSecrets[secretName]
		if !secretFound {
			return fmt.Errorf("finding secret %s in decrypted secret map", secretName)
		}

		if err := r.client.Create(ctx, secretToRestore); err != nil {
			return fmt.Errorf("recreating secret %s for central %s: %w", secretName, central.Id, err)
		}

	}

	return nil
}

func (r *CentralReconciler) reconcileAdminPasswordGeneration(central *v1alpha1.Central) error {
	if !r.wantsAuthProvider {
		central.Spec.Central.AdminPasswordGenerationDisabled = pointer.Bool(false)
		glog.Infof("No auth provider desired, enabling basic authentication for Central %s/%s",
			central.GetNamespace(), central.GetName())
		return nil
	}
	central.Spec.Central.AdminPasswordGenerationDisabled = pointer.Bool(true)
	return nil
}

func (r *CentralReconciler) ensureAuthProviderExists(ctx context.Context, remoteCentral private.ManagedCentral) (bool, error) {
	// Short-circuit if an auth provider isn't desired or already exists.
	if !r.wantsAuthProvider {
		return true, nil
	}

	exists, err := r.verifyAuthProviderFunc(ctx, remoteCentral, r.client)
	if err != nil {
		return false, errors.Wrapf(err, "failed to verify that the default auth provider exists within "+
			"Central %s/%s", remoteCentral.Metadata.Namespace, remoteCentral.Metadata.Name)
	}
	if exists {
		r.hasAuthProvider = true
		return true, nil
	}
	return false, nil
}

func (r *CentralReconciler) reconcileInstanceDeletion(ctx context.Context, remoteCentral *private.ManagedCentral, central *v1alpha1.Central) (*private.DataPlaneCentralStatus, error) {
	remoteCentralName := remoteCentral.Metadata.Name
	remoteCentralNamespace := remoteCentral.Metadata.Namespace

	deleted, err := r.ensureCentralDeleted(ctx, remoteCentral, central)
	if err != nil {
		return nil, errors.Wrapf(err, "delete central %s/%s", remoteCentralNamespace, remoteCentralName)
	}
	if deleted {
		return deletedStatus(), nil
	}
	return nil, ErrDeletionInProgress
}

func (r *CentralReconciler) reconcileCentralDBConfig(ctx context.Context, remoteCentral *private.ManagedCentral, central *v1alpha1.Central) error {

	if central.Spec.Central == nil {
		central.Spec.Central = &v1alpha1.CentralComponentSpec{}
	}
	if central.Spec.Central.DB == nil {
		central.Spec.Central.DB = &v1alpha1.CentralDBSpec{}
	}
	central.Spec.Central.DB.IsEnabled = v1alpha1.CentralDBEnabledPtr(v1alpha1.CentralDBEnabledTrue)

	if !r.managedDBEnabled {
		return nil
	}

	centralDBConnectionString, err := r.getCentralDBConnectionString(ctx, remoteCentral)
	if err != nil {
		return fmt.Errorf("getting Central DB connection string: %w", err)
	}

	central.Spec.Central.DB.ConnectionStringOverride = pointer.String(centralDBConnectionString)
	central.Spec.Central.DB.PasswordSecret = &v1alpha1.LocalSecretReference{
		Name: centralDbSecretName,
	}

	dbCA, err := postgres.GetDatabaseCACertificates()
	if err != nil {
		glog.Warningf("Could not read DB server CA bundle: %v", err)
	} else {
		central.Spec.TLS = &v1alpha1.TLSConfig{
			AdditionalCAs: []v1alpha1.AdditionalCA{
				{
					Name:    postgres.CentralDatabaseCACertificateBaseName,
					Content: string(dbCA),
				},
			},
		}
	}
	return nil
}

func getAuditLogNotifierConfig(
	auditLoggingConfig config.AuditLogging,
	namespace string,
) *declarativeconfig.Notifier {
	return &declarativeconfig.Notifier{
		Name: auditLogNotifierName,
		GenericConfig: &declarativeconfig.GenericConfig{
			Endpoint:            auditLoggingConfig.Endpoint(true),
			SkipTLSVerify:       auditLoggingConfig.SkipTLSVerify,
			AuditLoggingEnabled: true,
			ExtraFields: []declarativeconfig.KeyValuePair{
				{
					Key:   auditLogTenantIDKey,
					Value: namespace,
				},
			},
		},
	}
}

func (r *CentralReconciler) configureAuditLogNotifier(secret *corev1.Secret, namespace string) error {
	if !r.auditLogging.Enabled {
		return nil
	}
	if secret.Data == nil {
		secret.Data = make(map[string][]byte)
	}
	auditLogNotifierConfig := getAuditLogNotifierConfig(
		r.auditLogging,
		namespace,
	)
	encodedNotifierConfig, marshalErr := yaml.Marshal(auditLogNotifierConfig)
	if marshalErr != nil {
		return fmt.Errorf("marshaling audit log notifier configuration: %w", marshalErr)
	}
	secret.Data[auditLogNotifierKey] = encodedNotifierConfig
	return nil
}

func getAuthProviderConfig(remoteCentral private.ManagedCentral) *declarativeconfig.AuthProvider {
	groups := []declarativeconfig.Group{
		{
			AttributeKey:   "userid",
			AttributeValue: remoteCentral.Spec.Auth.OwnerUserId,
			RoleName:       "Admin",
		},
		{
			AttributeKey:   "groups",
			AttributeValue: "admin:org:all",
			RoleName:       "Admin",
		},
		{
			AttributeKey:   "rh_is_org_admin",
			AttributeValue: "true",
			RoleName:       "Admin",
		},
	}
	if remoteCentral.Spec.Auth.OwnerAlternateUserId != "" {
		groups = append(groups, declarativeconfig.Group{
			AttributeKey:   "userid",
			AttributeValue: remoteCentral.Spec.Auth.OwnerAlternateUserId,
			RoleName:       "Admin",
		})
	}
	return &declarativeconfig.AuthProvider{
		Name:             authProviderName(remoteCentral),
		UIEndpoint:       remoteCentral.Spec.UiEndpoint.Host,
		ExtraUIEndpoints: []string{"localhost:8443"},
		Groups:           groups,
		RequiredAttributes: []declarativeconfig.RequiredAttribute{
			{
				AttributeKey:   "rh_org_id",
				AttributeValue: remoteCentral.Spec.Auth.OwnerOrgId,
			},
		},
		ClaimMappings: []declarativeconfig.ClaimMapping{
			{
				Path: "realm_access.roles",
				Name: "groups",
			},
			{
				Path: "org_id",
				Name: "rh_org_id",
			},
			{
				Path: "is_org_admin",
				Name: "rh_is_org_admin",
			},
		},
		OIDCConfig: &declarativeconfig.OIDCConfig{
			Issuer:                    remoteCentral.Spec.Auth.Issuer,
			CallbackMode:              "post",
			ClientID:                  remoteCentral.Spec.Auth.ClientId,
			ClientSecret:              remoteCentral.Spec.Auth.ClientSecret, // pragma: allowlist secret
			DisableOfflineAccessScope: true,
		},
	}
}

func (r *CentralReconciler) configureAuthProvider(secret *corev1.Secret, remoteCentral private.ManagedCentral) error {
	if !r.wantsAuthProvider {
		return nil
	}

	if secret.Data == nil {
		secret.Data = make(map[string][]byte)
	}

	authProviderConfig := getAuthProviderConfig(remoteCentral)

	rawAuthProviderBytes, err := yaml.Marshal(authProviderConfig)
	if err != nil {
		return fmt.Errorf("marshaling auth provider configuration: %w", err)
	}
	secret.Data[authProviderDeclarativeConfigKey] = rawAuthProviderBytes
	return nil
}

func (r *CentralReconciler) reconcileDeclarativeConfigurationData(ctx context.Context,
	remoteCentral private.ManagedCentral) error {
	namespace := remoteCentral.Metadata.Namespace
	return r.ensureSecretExists(
		ctx,
		namespace,
		sensibleDeclarativeConfigSecretName,
		func(secret *corev1.Secret) error {
			var configErrs *multierror.Error
			if err := r.configureAuditLogNotifier(secret, namespace); err != nil {
				configErrs = multierror.Append(configErrs, err)
			}
			if err := r.configureAuthProvider(secret, remoteCentral); err != nil {
				configErrs = multierror.Append(configErrs, err)
			}
			if err := r.configureAdditionalAuthProvider(secret, remoteCentral); err != nil {
				configErrs = multierror.Append(configErrs, err)
			}
			return errors.Wrapf(configErrs.ErrorOrNil(),
				"configuring declarative configurations within secret %s/%s",
				secret.GetNamespace(), secret.GetName())
		},
	)
}

func (r *CentralReconciler) reconcileCentral(ctx context.Context, remoteCentral *private.ManagedCentral, central *v1alpha1.Central) error {
	remoteCentralName := remoteCentral.Metadata.Name
	remoteCentralNamespace := remoteCentral.Metadata.Namespace

	centralExists := true
	existingCentral := v1alpha1.Central{}
	centralKey := ctrlClient.ObjectKey{Namespace: remoteCentralNamespace, Name: remoteCentralName}
	err := r.client.Get(ctx, centralKey, &existingCentral)
	if err != nil {
		if !apiErrors.IsNotFound(err) {
			return errors.Wrapf(err, "unable to check the existence of central %v", centralKey)
		}
		centralExists = false
	}

	if remoteCentral.Metadata.ExpiredAt != nil {
		if central.GetAnnotations() == nil {
			central.Annotations = map[string]string{}
		}
		central.Annotations[centralExpiredAtKey] = remoteCentral.Metadata.ExpiredAt.Format(time.RFC3339)
	}

	if !centralExists {
		if central.GetAnnotations() == nil {
			central.Annotations = map[string]string{}
		}
		if err := util.IncrementCentralRevision(central); err != nil {
			return errors.Wrapf(err, "incrementing Central %v revision", centralKey)
		}

		glog.Infof("Creating Central %v", centralKey)
		if err := r.client.Create(ctx, central); err != nil {
			return errors.Wrapf(err, "creating new Central %v", centralKey)
		}
		glog.Infof("Central %v created", centralKey)
	} else {
		// perform a dry run to see if the update would change anything.
		// This would apply the defaults and the mutating webhooks without actually updating the object.
		// We can then compare the existing object with the object that would be resulting from the update.
		// This will prevent unnecessary operator reconciliation loops.

		desiredCentral := existingCentral.DeepCopy()
		desiredCentral.Spec = *central.Spec.DeepCopy()
		mergeLabelsAndAnnotations(central, desiredCentral)

		requiresUpdate, err := centralNeedsUpdating(ctx, r.client, &existingCentral, desiredCentral)
		if err != nil {
			return errors.Wrapf(err, "checking if Central %v needs to be updated", centralKey)
		}

		if !requiresUpdate {
			glog.Infof("Central %v is already up to date", centralKey)
			return nil
		}

		if err := util.IncrementCentralRevision(desiredCentral); err != nil {
			return errors.Wrapf(err, "incrementing Central %v revision", centralKey)
		}

		if err := r.client.Update(context.Background(), desiredCentral); err != nil {
			return errors.Wrapf(err, "updating Central %v", centralKey)
		}

	}

	return nil
}

func mergeLabelsAndAnnotations(from, into *v1alpha1.Central) {
	if into.Annotations == nil {
		into.Annotations = map[string]string{}
	}
	if into.Labels == nil {
		into.Labels = map[string]string{}
	}
	into.Annotations = mergeStringsMap(from.Annotations, into.Annotations)
	into.Labels = mergeStringsMap(from.Labels, into.Labels)
}

func mergeStringsMap(from, into map[string]string) map[string]string {
	var result = map[string]string{}
	for key, value := range into {
		result[key] = value
	}
	for key, value := range from {
		result[key] = value
	}
	return result
}

func centralNeedsUpdating(ctx context.Context, client ctrlClient.Client, existing *v1alpha1.Central, desired *v1alpha1.Central) (bool, error) {
	wouldBeCentral := desired.DeepCopy()
	centralKey := ctrlClient.ObjectKey{Namespace: existing.Namespace, Name: existing.Name}
	if err := client.Update(ctx, desired, ctrlClient.DryRunAll); err != nil {
		return false, errors.Wrapf(err, "dry-run updating Central %v", centralKey)
	}

	var shouldUpdate = false
	if !reflect.DeepEqual(existing.Spec, wouldBeCentral.Spec) {
		glog.Infof("Detected that Central %v is out of date and needs to be updated", centralKey)
		shouldUpdate = true
	}

	if !shouldUpdate && stringMapNeedsUpdating(desired.Annotations, existing.Annotations) {
		glog.Infof("Detected that Central %v annotations are out of date and needs to be updated", centralKey)
		shouldUpdate = true
	}

	if !shouldUpdate && stringMapNeedsUpdating(desired.Labels, existing.Labels) {
		glog.Infof("Detected that Central %v labels are out of date and needs to be updated", centralKey)
		shouldUpdate = true
	}

	if shouldUpdate {
		printCentralDiff(wouldBeCentral, existing)
	}

	return shouldUpdate, nil
}

func stringMapNeedsUpdating(desired, actual map[string]string) bool {
	if len(desired) == 0 {
		return false
	}
	if actual == nil {
		return true
	}
	for k, v := range desired {
		if actual[k] != v {
			return true
		}
	}
	return false
}

func printCentralDiff(desired, actual *v1alpha1.Central) {
	if !features.PrintCentralUpdateDiff.Enabled() {
		return
	}
	desiredBytes, err := json.Marshal(desired.Spec)
	if err != nil {
		glog.Warningf("Failed to marshal desired Central %s/%s spec: %v", desired.Namespace, desired.Name, err)
		return
	}
	actualBytes, err := json.Marshal(actual.Spec)
	if err != nil {
		glog.Warningf("Failed to marshal actual Central %s/%s spec: %v", desired.Namespace, desired.Name, err)
		return
	}
	patchBytes, err := strategicpatch.CreateTwoWayMergePatch(actualBytes, desiredBytes, &v1alpha1.CentralSpec{})
	if err != nil {
		glog.Warningf("Failed to create Central %s/%s patch: %v", desired.Namespace, desired.Name, err)
		return
	}
	glog.Infof("Central %s/%s diff: %s", desired.Namespace, desired.Name, string(patchBytes))
}

func (r *CentralReconciler) collectReconciliationStatus(ctx context.Context, remoteCentral *private.ManagedCentral) (*private.DataPlaneCentralStatus, error) {
	remoteCentralNamespace := remoteCentral.Metadata.Namespace

	status := readyStatus()
	// Do not report routes statuses if:
	// 1. Routes are not used on the cluster
	// 2. Central request is in status "Ready" - assuming that routes are already reported and saved
	if r.useRoutes && !isRemoteCentralReady(remoteCentral) {
		var err error
		status.Routes, err = r.getRoutesStatuses(ctx, remoteCentralNamespace)
		if err != nil {
			return nil, err
		}
	}

	// Only report secrets if Central is ready, to ensure we're not trying to get secrets before they are created.
	// Only report secrets if not all secrets are already stored to ensure we don't overwrite initial secrets with corrupted secrets
	// from the cluster state.
	if isRemoteCentralReady(remoteCentral) && !r.areSecretsStored(remoteCentral.Metadata.SecretsStored) {
		secrets, err := r.collectSecretsEncrypted(ctx, remoteCentral)
		if err != nil {
			return nil, err
		}
		status.Secrets = secrets // pragma: allowlist secret
	}

	return status, nil
}

func (r *CentralReconciler) areSecretsStored(secretsStored []string) bool {
	secretsStoredSize := len(secretsStored)
	expectedSecrets := r.secretBackup.GetWatchedSecrets()
	if secretsStoredSize != len(expectedSecrets) {
		return false
	}

	secretsStoredCopy := make([]string, secretsStoredSize)
	copy(secretsStoredCopy, secretsStored)
	sort.Strings(secretsStoredCopy)

	for i := 0; i < secretsStoredSize; i++ {
		if secretsStoredCopy[i] != expectedSecrets[i] {
			return false
		}
	}

	return true
}

func (r *CentralReconciler) collectSecrets(ctx context.Context, remoteCentral *private.ManagedCentral) (map[string]*corev1.Secret, error) {
	namespace := remoteCentral.Metadata.Namespace
	secrets, err := r.secretBackup.CollectSecrets(ctx, namespace)
	if err != nil {
		return secrets, fmt.Errorf("collecting secrets for namespace %s: %w", namespace, err)
	}

	// remove ResourceVersion and owner reference as this is only intended to recreate non-existent
	// resources instead of updating existing ones, the owner reference might get invalid in case of
	// central namespace recreation
	for _, secret := range secrets { // pragma: allowlist secret
		secret.ObjectMeta.ResourceVersion = ""
		secret.ObjectMeta.OwnerReferences = []metav1.OwnerReference{}
	}

	return secrets, nil
}

func (r *CentralReconciler) collectSecretsEncrypted(ctx context.Context, remoteCentral *private.ManagedCentral) (map[string]string, error) {
	secrets, err := r.collectSecrets(ctx, remoteCentral)
	if err != nil {
		return nil, err
	}

	encryptedSecrets, err := r.encryptSecrets(secrets)
	if err != nil {
		return nil, fmt.Errorf("encrypting secrets for namespace: %s: %w", remoteCentral.Metadata.Namespace, err)
	}

	return encryptedSecrets, nil
}

func (r *CentralReconciler) decryptSecrets(secrets map[string]string) (map[string]*corev1.Secret, error) {
	decryptedSecrets := map[string]*corev1.Secret{}

	for secretName, ciphertext := range secrets {
		decodedCipher, err := base64.StdEncoding.DecodeString(ciphertext)
		if err != nil {
			return nil, fmt.Errorf("decoding secret %s: %w", secretName, err)
		}

		plaintextSecret, err := r.secretCipher.Decrypt(decodedCipher)
		if err != nil {
			return nil, fmt.Errorf("decrypting secret %s: %w", secretName, err)
		}

		var secret corev1.Secret
		if err := json.Unmarshal(plaintextSecret, &secret); err != nil {
			return nil, fmt.Errorf("unmarshaling secret %s: %w", secretName, err)
		}

		decryptedSecrets[secretName] = &secret // pragma: allowlist secret
	}

	return decryptedSecrets, nil
}

func (r *CentralReconciler) encryptSecrets(secrets map[string]*corev1.Secret) (map[string]string, error) {
	encryptedSecrets := map[string]string{}

	for key, secret := range secrets { // pragma: allowlist secret
		secretBytes, err := json.Marshal(secret)
		if err != nil {
			return nil, fmt.Errorf("error marshaling secret for encryption: %s: %w", key, err)
		}

		encryptedBytes, err := r.secretCipher.Encrypt(secretBytes)
		if err != nil {
			return nil, fmt.Errorf("encrypting secret: %s: %w", key, err)
		}

		encryptedSecrets[key] = base64.StdEncoding.EncodeToString(encryptedBytes)
	}

	return encryptedSecrets, nil

}

// ensureSecretHasOwnerReference is used to make sure the central-tls secret has it's
// owner reference properly set after a restore operation so that the automatic cert rotation
// in the operator is working
func (r *CentralReconciler) ensureSecretHasOwnerReference(ctx context.Context, secretName string, remoteCentral *private.ManagedCentral, central *v1alpha1.Central) error {
	secret, err := r.getSecret(remoteCentral.Metadata.Namespace, secretName)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			// no need to ensure correct owner reference if the secret doesn't exist
			return nil
		}
		return err
	}

	if len(secret.ObjectMeta.OwnerReferences) != 0 {
		return nil
	}

	centralCR := &v1alpha1.Central{}
	if err := r.client.Get(ctx, ctrlClient.ObjectKeyFromObject(central), centralCR); err != nil {
		return fmt.Errorf("getting current central CR from k8s: %w", err)
	}

	secret.OwnerReferences = []metav1.OwnerReference{
		*metav1.NewControllerRef(centralCR, v1alpha1.CentralGVK),
	}

	if err := r.client.Update(ctx, secret); err != nil {
		return fmt.Errorf("updating %s secret: %w", k8s.CentralTLSSecretName, err)
	}

	return nil
}

func (r *CentralReconciler) ensureDeclarativeConfigurationSecretCleaned(ctx context.Context, remoteCentralNamespace string) error {
	secret := &corev1.Secret{}
	secretKey := ctrlClient.ObjectKey{ // pragma: allowlist secret
		Namespace: remoteCentralNamespace,
		Name:      sensibleDeclarativeConfigSecretName,
	}

	err := r.client.Get(ctx, secretKey, secret)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	return r.client.Delete(ctx, secret)
}

func isRemoteCentralProvisioning(remoteCentral private.ManagedCentral) bool {
	return remoteCentral.RequestStatus == centralConstants.CentralRequestStatusProvisioning.String()
}

func isRemoteCentralReady(remoteCentral *private.ManagedCentral) bool {
	return remoteCentral.RequestStatus == centralConstants.CentralRequestStatusReady.String()
}

func (r *CentralReconciler) getRoutesStatuses(ctx context.Context, namespace string) ([]private.DataPlaneCentralStatusRoutes, error) {
	reencryptIngress, err := r.routeService.FindReencryptIngress(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("obtaining ingress for reencrypt route: %w", err)
	}
	passthroughIngress, err := r.routeService.FindPassthroughIngress(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("obtaining ingress for passthrough route: %w", err)
	}
	return []private.DataPlaneCentralStatusRoutes{
		getRouteStatus(reencryptIngress),
		getRouteStatus(passthroughIngress),
	}, nil
}

func getRouteStatus(ingress *openshiftRouteV1.RouteIngress) private.DataPlaneCentralStatusRoutes {
	return private.DataPlaneCentralStatusRoutes{
		Domain: ingress.Host,
		Router: ingress.RouterCanonicalHostname,
	}
}

func (r *CentralReconciler) ensureCentralDeleted(ctx context.Context, remoteCentral *private.ManagedCentral, central *v1alpha1.Central) (bool, error) {
	globalDeleted := true
	if r.useRoutes {
		reencryptRouteDeleted, err := r.ensureReencryptRouteDeleted(ctx, central.GetNamespace())
		if err != nil {
			return false, err
		}
		passthroughRouteDeleted, err := r.ensurePassthroughRouteDeleted(ctx, central.GetNamespace())
		if err != nil {
			return false, err
		}

		globalDeleted = globalDeleted && reencryptRouteDeleted && passthroughRouteDeleted
	}

	centralDeleted, err := r.ensureCentralCRDeleted(ctx, central)
	if err != nil {
		return false, err
	}
	globalDeleted = globalDeleted && centralDeleted

	if err := r.ensureDeclarativeConfigurationSecretCleaned(ctx, central.GetNamespace()); err != nil {
		return false, nil
	}

	if r.managedDBEnabled {
		// skip Snapshot for remoteCentral created by probe
		skipSnapshot := remoteCentral.Metadata.Internal

		databaseID, err := r.getDatabaseID(ctx, remoteCentral.Metadata.Namespace, remoteCentral.Id)
		if err != nil {
			return false, fmt.Errorf("getting DB ID: %w", err)
		}

		err = r.managedDBProvisioningClient.EnsureDBDeprovisioned(databaseID, skipSnapshot)
		if err != nil {
			if errors.Is(err, cloudprovider.ErrDBBackupInProgress) {
				glog.Infof("Can not delete Central DB for: %s, backup in progress", remoteCentral.Metadata.Namespace)
				return false, nil
			}

			return false, fmt.Errorf("deprovisioning DB: %v", err)
		}

		secretDeleted, err := r.ensureCentralDBSecretDeleted(ctx, central.GetNamespace())
		if err != nil {
			return false, err
		}
		globalDeleted = globalDeleted && secretDeleted
	}

	chartResourcesDeleted, err := r.ensureChartResourcesDeleted(ctx, remoteCentral)
	if err != nil {
		return false, err
	}
	globalDeleted = globalDeleted && chartResourcesDeleted

	nsDeleted, err := r.ensureNamespaceDeleted(ctx, central.GetNamespace())
	if err != nil {
		return false, err
	}
	globalDeleted = globalDeleted && nsDeleted
	return globalDeleted, nil
}

// getDatabaseID returns the cloud database ID for a central tenant.
// By default the database ID is equal to the centralID. It can be overridden by a ConfigMap.
func (r *CentralReconciler) getDatabaseID(ctx context.Context, remoteCentralNamespace, centralID string) (string, error) {
	configMap := &corev1.ConfigMap{}
	err := r.client.Get(ctx, ctrlClient.ObjectKey{Namespace: remoteCentralNamespace, Name: centralDbOverrideConfigMap}, configMap)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			return centralID, nil
		}

		return centralID, fmt.Errorf("getting central DB ID override ConfigMap: %w", err)
	}

	overrideValue, exists := configMap.Data["databaseID"]
	if exists {
		glog.Infof("The database ID for Central %s is overridden with: %s", centralID, overrideValue)
		return overrideValue, nil
	}

	glog.Infof("The %s ConfigMap exists but contains no databaseID field, using default: %s", centralDbOverrideConfigMap, centralID)
	return centralID, nil
}

// centralChanged compares the given central to the last central reconciled using a hash
func (r *CentralReconciler) centralChanged(currentHash [16]byte) bool {
	return !bytes.Equal(r.lastCentralHash[:], currentHash[:])
}

func (r *CentralReconciler) setLastCentralHash(currentHash [16]byte) {
	r.lastCentralHash = currentHash
}

func (r *CentralReconciler) computeCentralHash(central private.ManagedCentral) ([16]byte, error) {
	hash, err := util.MD5SumFromJSONStruct(&central)
	if err != nil {
		return [16]byte{}, fmt.Errorf("calculating MD5 from JSON: %w", err)
	}
	return hash, nil
}

func (r *CentralReconciler) getNamespace(name string) (*corev1.Namespace, error) {
	var namespace corev1.Namespace
	if err := r.client.Get(context.Background(), ctrlClient.ObjectKey{Name: name}, &namespace); err != nil {
		return nil, fmt.Errorf("getting namespace %q: %w", name, err)
	}
	return &namespace, nil
}

func (r *CentralReconciler) getSecret(namespaceName string, secretName string) (*corev1.Secret, error) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespaceName,
		},
	}
	err := r.client.Get(context.Background(), ctrlClient.ObjectKey{Namespace: namespaceName, Name: secretName}, secret)
	if err != nil {
		return nil, errors.Wrapf(err, "retrieving secret %s/%s", namespaceName, secretName)
	}
	return secret, nil
}

func (r *CentralReconciler) createImagePullSecret(ctx context.Context, namespaceName string, secretName string, imagePullSecretJSON []byte) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespaceName,
			Name:      secretName,
		},
		Type: "kubernetes.io/dockerconfigjson",
		Data: map[string][]byte{
			".dockerconfigjson": imagePullSecretJSON,
		},
	}

	if err := r.client.Create(ctx, secret); err != nil {
		return errors.Wrapf(err, "creating image pull secret %s/%s", namespaceName, secretName)
	}

	return nil
}

func (r *CentralReconciler) reconcileNamespace(ctx context.Context, c private.ManagedCentral) error {
	desiredNamespace := getDesiredNamespace(c)

	existingNamespace, err := r.getNamespace(desiredNamespace.Name)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			if err := r.client.Create(ctx, desiredNamespace); err != nil {
				return fmt.Errorf("creating namespace %q: %w", desiredNamespace.Name, err)
			}
			return nil
		}
		return fmt.Errorf("getting namespace %q: %w", desiredNamespace.Name, err)
	}

	if stringMapNeedsUpdating(desiredNamespace.Annotations, existingNamespace.Annotations) || stringMapNeedsUpdating(desiredNamespace.Labels, existingNamespace.Labels) {
		glog.Infof("Updating namespace %q", desiredNamespace.Name)
		if existingNamespace.Annotations == nil {
			existingNamespace.Annotations = map[string]string{}
		}
		for k, v := range desiredNamespace.Annotations {
			existingNamespace.Annotations[k] = v
		}
		if existingNamespace.Labels == nil {
			existingNamespace.Labels = map[string]string{}
		}
		for k, v := range desiredNamespace.Labels {
			existingNamespace.Labels[k] = v
		}
		if err = r.client.Update(ctx, existingNamespace, &ctrlClient.UpdateOptions{
			FieldManager: "fleetshard-sync",
		}); err != nil {
			return fmt.Errorf("updating namespace %q: %w", desiredNamespace.Name, err)
		}
	}

	return nil
}

func getDesiredNamespace(c private.ManagedCentral) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:        c.Metadata.Namespace,
			Annotations: getNamespaceAnnotations(c),
			Labels:      getNamespaceLabels(c),
		},
	}
}

func (r *CentralReconciler) ensureImagePullSecretConfigured(ctx context.Context, namespaceName string, secretName string, imagePullSecret []byte) error {
	// Ensure that the secret exists.
	_, err := r.getSecret(namespaceName, secretName)
	if err == nil {
		// Secret exists already.
		return nil
	}
	if !apiErrors.IsNotFound(err) {
		// Unexpected error.
		return errors.Wrapf(err, "retrieving secret %s/%s", namespaceName, secretName)
	}
	// We have an IsNotFound error.
	glog.Infof("Creating image pull secret %s/%s", namespaceName, secretName)
	return r.createImagePullSecret(ctx, namespaceName, secretName, imagePullSecret)
}

func (r *CentralReconciler) ensureNamespaceDeleted(ctx context.Context, name string) (bool, error) {
	namespace, err := r.getNamespace(name)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			return true, nil
		}
		return false, errors.Wrapf(err, "delete central namespace %s", name)
	}
	if namespace.Status.Phase == corev1.NamespaceTerminating {
		return false, nil // Deletion is already in progress, skipping deletion request
	}
	if err = r.client.Delete(ctx, namespace); err != nil {
		return false, errors.Wrapf(err, "delete central namespace %s", name)
	}
	glog.Infof("Central namespace %s is marked for deletion", name)
	return false, nil
}

func (r *CentralReconciler) ensureEncryptionKeySecretExists(ctx context.Context, remoteCentralNamespace string) error {
	return r.ensureSecretExists(ctx, remoteCentralNamespace, centralEncryptionKeySecretName, r.populateEncryptionKeySecret)
}

func (r *CentralReconciler) populateEncryptionKeySecret(secret *corev1.Secret) error {
	const encryptionKeyChainFile = "key-chain.yaml"

	if secret.Data != nil {
		if _, ok := secret.Data[encryptionKeyChainFile]; ok {
			// secret already populated with encryption key skip operation
			return nil
		}
	}

	encryptionKey, err := r.encryptionKeyGenerator.Generate()
	if err != nil {
		return fmt.Errorf("generating encryption key: %w", err)
	}

	b64Key := base64.StdEncoding.EncodeToString(encryptionKey)
	keyChainFile, err := generateNewKeyChainFile(b64Key)
	if err != nil {
		return err
	}
	secret.Data = map[string][]byte{encryptionKeyChainFile: keyChainFile}
	return nil
}

func generateNewKeyChainFile(b64Key string) ([]byte, error) {
	keyMap := make(map[int]string)
	keyMap[0] = b64Key

	keyChain := centralNotifierUtils.KeyChain{
		KeyMap:         keyMap,
		ActiveKeyIndex: 0,
	}

	yamlBytes, err := yaml.Marshal(keyChain)
	if err != nil {
		return []byte{}, fmt.Errorf("generating key-chain file: %w", err)
	}

	return yamlBytes, nil
}

func (r *CentralReconciler) getCentralDBConnectionString(ctx context.Context, remoteCentral *private.ManagedCentral) (string, error) {
	centralDBUserExists, err := r.centralDBUserExists(ctx, remoteCentral.Metadata.Namespace)
	if err != nil {
		return "", err
	}

	// If a Central DB user already exists, it means the managed DB was already
	// provisioned and successfully created (access to a running Postgres instance is a
	// precondition to create this user)
	if !centralDBUserExists {
		if err := r.ensureManagedCentralDBInitialized(ctx, remoteCentral); err != nil {
			return "", fmt.Errorf("initializing managed DB: %w", err)
		}
	}

	databaseID, err := r.getDatabaseID(ctx, remoteCentral.Metadata.Namespace, remoteCentral.Id)
	if err != nil {
		return "", fmt.Errorf("getting DB ID: %w", err)
	}

	dbConnection, err := r.managedDBProvisioningClient.GetDBConnection(databaseID)
	if err != nil {
		if !errors.Is(err, cloudprovider.ErrDBNotFound) {
			return "", fmt.Errorf("getting RDS DB connection data: %w", err)
		}

		glog.Infof("expected DB for %s not found, trying to restore...", remoteCentral.Id)
		// Using no password because we try to restore from backup
		err := r.managedDBProvisioningClient.EnsureDBProvisioned(ctx, remoteCentral.Id, remoteCentral.Id, "", remoteCentral.Metadata.Internal)
		if err != nil {
			return "", fmt.Errorf("trying to restore DB: %w", err)
		}
	}

	return dbConnection.GetConnectionForUserAndDB(dbCentralUserName, postgres.CentralDBName).WithSSLRootCert(postgres.DatabaseCACertificatePathCentral).AsConnectionString(), nil
}

func generateDBPassword() (string, error) {
	password, err := random.GenerateString(25, random.AlphanumericCharacters)
	if err != nil {
		return "", fmt.Errorf("generating DB password: %w", err)
	}

	return password, nil
}

func (r *CentralReconciler) ensureManagedCentralDBInitialized(ctx context.Context, remoteCentral *private.ManagedCentral) error {
	remoteCentralNamespace := remoteCentral.Metadata.Namespace

	centralDBSecretExists, err := r.centralDBSecretExists(ctx, remoteCentralNamespace)
	if err != nil {
		return err
	}

	if !centralDBSecretExists {
		dbMasterPassword, err := generateDBPassword()
		if err != nil {
			return fmt.Errorf("generating Central DB master password: %w", err)
		}
		if err := r.ensureCentralDBSecretExists(ctx, remoteCentralNamespace, dbUserTypeMaster, dbMasterPassword); err != nil {
			return fmt.Errorf("ensuring that DB secret exists: %w", err)
		}
	}

	dbMasterPassword, err := r.getDBPasswordFromSecret(ctx, remoteCentralNamespace)
	if err != nil {
		return fmt.Errorf("getting DB password from secret: %w", err)
	}

	databaseID, err := r.getDatabaseID(ctx, remoteCentralNamespace, remoteCentral.Id)
	if err != nil {
		return fmt.Errorf("getting DB ID: %w", err)
	}

	err = r.managedDBProvisioningClient.EnsureDBProvisioned(ctx, databaseID, remoteCentral.Id, dbMasterPassword, remoteCentral.Metadata.Internal)
	if err != nil {
		return fmt.Errorf("provisioning RDS DB: %w", err)
	}

	dbConnection, err := r.managedDBProvisioningClient.GetDBConnection(databaseID)
	if err != nil {
		return fmt.Errorf("getting RDS DB connection data: %w", err)
	}

	dbCentralPassword, err := generateDBPassword()
	if err != nil {
		return fmt.Errorf("generating Central DB password: %w", err)
	}
	err = r.managedDBInitFunc(ctx, dbConnection.WithPassword(dbMasterPassword).WithSSLRootCert(postgres.DatabaseCACertificatePathFleetshard),
		dbCentralUserName, dbCentralPassword)
	if err != nil {
		return fmt.Errorf("initializing managed DB: %w", err)
	}

	// Replace the password stored in the secret. This replaces the master password (the password of the
	// rds_superuser account) with the password of the Central user. Note that we don't store
	// the master password anywhere from this point on.
	err = r.ensureCentralDBSecretExists(ctx, remoteCentralNamespace, dbUserTypeCentral, dbCentralPassword)
	if err != nil {
		return err
	}

	return nil
}

func (r *CentralReconciler) ensureCentralDBSecretExists(ctx context.Context, remoteCentralNamespace, userType, password string) error {
	setPasswordFunc := func(secret *corev1.Secret, userType, password string) {
		secret.Data = map[string][]byte{"password": []byte(password)}
		if secret.Annotations == nil {
			secret.Annotations = make(map[string]string)
		}
		secret.Annotations[dbUserTypeAnnotation] = userType
	}
	return r.ensureSecretExists(ctx, remoteCentralNamespace, centralDbSecretName, func(secret *corev1.Secret) error {
		setPasswordFunc(secret, userType, password)
		return nil
	})
}

func (r *CentralReconciler) centralDBSecretExists(ctx context.Context, remoteCentralNamespace string) (bool, error) {
	return r.checkSecretExists(ctx, remoteCentralNamespace, centralDbSecretName)
}

func (r *CentralReconciler) centralDBUserExists(ctx context.Context, remoteCentralNamespace string) (bool, error) {
	secret := &corev1.Secret{}
	err := r.client.Get(ctx, ctrlClient.ObjectKey{Namespace: remoteCentralNamespace, Name: centralDbSecretName}, secret)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			return false, nil
		}

		return false, fmt.Errorf("getting central DB secret: %w", err)
	}

	if secret.Annotations == nil {
		// if the annotation section is missing, assume it's the master password
		return false, nil
	}

	dbUserType, exists := secret.Annotations[dbUserTypeAnnotation]
	if !exists {
		// legacy Centrals use the master password and do not have this annotation
		return false, nil
	}

	return dbUserType == dbUserTypeCentral, nil
}

func (r *CentralReconciler) ensureCentralDBSecretDeleted(ctx context.Context, remoteCentralNamespace string) (bool, error) {
	secret := &corev1.Secret{}
	err := r.client.Get(ctx, ctrlClient.ObjectKey{Namespace: remoteCentralNamespace, Name: centralDbSecretName}, secret)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			return true, nil
		}

		return false, fmt.Errorf("deleting Central DB secret: %w", err)
	}

	if err := r.client.Delete(ctx, secret); err != nil {
		return false, fmt.Errorf("deleting central DB secret %s/%s", remoteCentralNamespace, centralDbSecretName)
	}

	glog.Infof("Central DB secret %s/%s is marked for deletion", remoteCentralNamespace, centralDbSecretName)
	return false, nil
}

func (r *CentralReconciler) getDBPasswordFromSecret(ctx context.Context, centralNamespace string) (string, error) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: centralDbSecretName,
		},
	}
	err := r.client.Get(ctx, ctrlClient.ObjectKey{Namespace: centralNamespace, Name: centralDbSecretName}, secret)
	if err != nil {
		return "", fmt.Errorf("getting Central DB password from secret: %w", err)
	}

	if dbPassword, ok := secret.Data["password"]; ok {
		return string(dbPassword), nil
	}

	return "", fmt.Errorf("central DB secret does not contain password field: %w", err)
}

func (r *CentralReconciler) ensureCentralCRDeleted(ctx context.Context, central *v1alpha1.Central) (bool, error) {
	centralKey := ctrlClient.ObjectKey{
		Namespace: central.GetNamespace(),
		Name:      central.GetName(),
	}

	err := wait.PollUntilContextCancel(ctx, centralDeletePollInterval, true, func(ctx context.Context) (bool, error) {
		var centralToDelete v1alpha1.Central

		if err := r.client.Get(ctx, centralKey, &centralToDelete); err != nil {
			if apiErrors.IsNotFound(err) {
				return true, nil
			}
			return false, errors.Wrapf(err, "failed to get central CR %v", centralKey)
		}

		// avoid being stuck in a deprovisioning state due to the pause reconcile annotation
		if err := r.disablePauseReconcileIfPresent(ctx, &centralToDelete); err != nil {
			return false, err
		}

		if centralToDelete.GetDeletionTimestamp() == nil {
			glog.Infof("Marking Central CR %v for deletion", centralKey)
			if err := r.client.Delete(ctx, &centralToDelete); err != nil {
				if apiErrors.IsNotFound(err) {
					return true, nil
				}
				return false, errors.Wrapf(err, "failed to delete central CR %v", centralKey)
			}
		}

		glog.Infof("Waiting for Central CR %v to be deleted", centralKey)
		return false, nil
	})

	if err != nil {
		return false, errors.Wrapf(err, "waiting for central CR %v to be deleted", centralKey)
	}
	glog.Infof("Central CR %v is deleted", centralKey)
	return true, nil
}

func (r *CentralReconciler) disablePauseReconcileIfPresent(ctx context.Context, central *v1alpha1.Central) error {
	if central.Annotations == nil {
		return nil
	}

	if value, exists := central.Annotations[PauseReconcileAnnotation]; !exists || value != "true" {
		return nil
	}

	central.Annotations[PauseReconcileAnnotation] = "false"
	err := r.client.Update(ctx, central)
	if err != nil {
		return fmt.Errorf("removing pause reconcile annotation: %v", err)
	}

	return nil
}

func (r *CentralReconciler) ensureChartResourcesExist(ctx context.Context, remoteCentral private.ManagedCentral) error {
	vals, err := r.chartValues(remoteCentral)
	if err != nil {
		return fmt.Errorf("obtaining values for resources chart: %w", err)
	}

	objs, err := charts.RenderToObjects(helmReleaseName, remoteCentral.Metadata.Namespace, r.resourcesChart, vals)
	if err != nil {
		return fmt.Errorf("rendering resources chart: %w", err)
	}
	for _, obj := range objs {
		if obj.GetNamespace() == "" {
			obj.SetNamespace(remoteCentral.Metadata.Namespace)
		}
		err := charts.InstallOrUpdateChart(ctx, obj, r.client)
		if err != nil {
			return fmt.Errorf("failed to update central tenant object %w", err)
		}
	}

	return nil
}

func (r *CentralReconciler) ensureChartResourcesDeleted(ctx context.Context, remoteCentral *private.ManagedCentral) (bool, error) {
	vals, err := r.chartValues(*remoteCentral)
	if err != nil {
		return false, fmt.Errorf("obtaining values for resources chart: %w", err)
	}

	objs, err := charts.RenderToObjects(helmReleaseName, remoteCentral.Metadata.Namespace, r.resourcesChart, vals)
	if err != nil {
		return false, fmt.Errorf("rendering resources chart: %w", err)
	}

	waitForDelete := false
	for _, obj := range objs {
		key := ctrlClient.ObjectKey{Namespace: obj.GetNamespace(), Name: obj.GetName()}
		if key.Namespace == "" {
			key.Namespace = remoteCentral.Metadata.Namespace
		}
		var out unstructured.Unstructured
		out.SetGroupVersionKind(obj.GroupVersionKind())
		err := r.client.Get(ctx, key, &out)
		if err != nil {
			if apiErrors.IsNotFound(err) {
				continue
			}
			return false, fmt.Errorf("retrieving object %s/%s of type %v: %w", key.Namespace, key.Name, obj.GroupVersionKind(), err)
		}
		if out.GetDeletionTimestamp() != nil {
			waitForDelete = true
			continue
		}
		err = r.client.Delete(ctx, &out)
		if err != nil && !apiErrors.IsNotFound(err) {
			return false, fmt.Errorf("retrieving object %s/%s of type %v: %w", key.Namespace, key.Name, obj.GroupVersionKind(), err)
		}
	}
	return !waitForDelete, nil
}

func (r *CentralReconciler) ensureRoutesExist(ctx context.Context, remoteCentral private.ManagedCentral) error {
	err := r.ensureReencryptRouteExists(ctx, remoteCentral)
	if err != nil {
		return err
	}
	return r.ensurePassthroughRouteExists(ctx, remoteCentral)
}

// TODO(ROX-9310): Move re-encrypt route reconciliation to the StackRox operator
func (r *CentralReconciler) ensureReencryptRouteExists(ctx context.Context, remoteCentral private.ManagedCentral) error {
	namespace := remoteCentral.Metadata.Namespace
	route, err := r.routeService.FindReencryptRoute(ctx, namespace)
	if err != nil && !apiErrors.IsNotFound(err) {
		return fmt.Errorf("retrieving reencrypt route for namespace %q: %w", namespace, err)
	}

	if apiErrors.IsNotFound(err) {
		err = r.routeService.CreateReencryptRoute(ctx, remoteCentral)
		if err != nil {
			return fmt.Errorf("creating reencrypt route for central %s: %w", remoteCentral.Id, err)
		}

		return nil
	}

	err = r.routeService.UpdateReencryptRoute(ctx, route, remoteCentral)
	if err != nil {
		return fmt.Errorf("updating reencrypt route for central %s: %w", remoteCentral.Id, err)
	}

	return nil
}

type routeSupplierFunc func() (*openshiftRouteV1.Route, error)

// TODO(ROX-9310): Move re-encrypt route reconciliation to the StackRox operator
// TODO(ROX-11918): Make hostname configurable on the StackRox operator
func (r *CentralReconciler) ensureReencryptRouteDeleted(ctx context.Context, namespace string) (bool, error) {
	return r.ensureRouteDeleted(ctx, func() (*openshiftRouteV1.Route, error) {
		return r.routeService.FindReencryptRoute(ctx, namespace) //nolint:wrapcheck
	})
}

// TODO(ROX-11918): Make hostname configurable on the StackRox operator
func (r *CentralReconciler) ensurePassthroughRouteExists(ctx context.Context, remoteCentral private.ManagedCentral) error {
	namespace := remoteCentral.Metadata.Namespace
	route, err := r.routeService.FindPassthroughRoute(ctx, namespace)
	if err != nil && !apiErrors.IsNotFound(err) {
		return fmt.Errorf("retrieving passthrough route for namespace %q: %w", namespace, err)
	}

	if apiErrors.IsNotFound(err) {
		err = r.routeService.CreatePassthroughRoute(ctx, remoteCentral)
		if err != nil {
			return fmt.Errorf("creating passthrough route for central %s: %w", remoteCentral.Id, err)
		}

		return nil
	}

	err = r.routeService.UpdatePassthroughRoute(ctx, route, remoteCentral)
	if err != nil {
		return fmt.Errorf("updating passthrough route for central %s: %w", remoteCentral.Id, err)
	}

	return nil
}

// TODO(ROX-11918): Make hostname configurable on the StackRox operator
func (r *CentralReconciler) ensurePassthroughRouteDeleted(ctx context.Context, namespace string) (bool, error) {
	return r.ensureRouteDeleted(ctx, func() (*openshiftRouteV1.Route, error) {
		return r.routeService.FindPassthroughRoute(ctx, namespace) //nolint:wrapcheck
	})
}

func (r *CentralReconciler) ensureRouteDeleted(ctx context.Context, routeSupplier routeSupplierFunc) (bool, error) {
	route, err := routeSupplier()
	if err != nil {
		if apiErrors.IsNotFound(err) {
			return true, nil
		}
		return false, errors.Wrapf(err, "get central route %s/%s", route.GetNamespace(), route.GetName())
	}
	if err := r.client.Delete(ctx, route); err != nil {
		return false, errors.Wrapf(err, "delete central route %s/%s", route.GetNamespace(), route.GetName())
	}
	return false, nil
}

func getTenantLabels(c private.ManagedCentral) map[string]string {
	return map[string]string{
		managedByLabelKey:    labelManagedByFleetshardValue,
		instanceLabelKey:     c.Metadata.Name,
		orgIDLabelKey:        c.Spec.Auth.OwnerOrgId,
		tenantIDLabelKey:     c.Id,
		instanceTypeLabelKey: c.Spec.InstanceType,
	}
}

func getTenantAnnotations(c private.ManagedCentral) map[string]string {
	return map[string]string{
		orgNameAnnotationKey: c.Spec.Auth.OwnerOrgName,
	}
}

func getNamespaceLabels(c private.ManagedCentral) map[string]string {
	return getTenantLabels(c)
}

func getNamespaceAnnotations(c private.ManagedCentral) map[string]string {
	namespaceAnnotations := getTenantAnnotations(c)
	if c.Metadata.ExpiredAt != nil {
		namespaceAnnotations[centralExpiredAtKey] = c.Metadata.ExpiredAt.Format(time.RFC3339)
	}
	return namespaceAnnotations
}

func (r *CentralReconciler) chartValues(c private.ManagedCentral) (chartutil.Values, error) {
	if r.resourcesChart == nil {
		return nil, errors.New("resources chart is not set")
	}
	src := r.resourcesChart.Values
	dst := map[string]interface{}{
		"labels":      stringMapToMapInterface(getTenantLabels(c)),
		"annotations": stringMapToMapInterface(getTenantAnnotations(c)),
	}
	if r.egressProxyImage != "" {
		dst["egressProxy"] = map[string]interface{}{
			"image": r.egressProxyImage,
		}
	}
	return chartutil.CoalesceTables(dst, src), nil
}

func stringMapToMapInterface(m map[string]string) map[string]interface{} {
	result := make(map[string]interface{}, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

func (r *CentralReconciler) shouldSkipReadyCentral(remoteCentral private.ManagedCentral) bool {
	return r.wantsAuthProvider == r.hasAuthProvider &&
		isRemoteCentralReady(&remoteCentral)
}

func (r *CentralReconciler) needsReconcile(changed bool, central *v1alpha1.Central, storedSecrets []string) bool {
	if !r.areSecretsStoredFunc(storedSecrets) {
		return true
	}

	if changed {
		return true
	}

	if r.clock.Now().Sub(r.lastCentralHashTime) > time.Minute*15 {
		return true
	}

	forceReconcile, ok := central.Labels["rhacs.redhat.com/force-reconcile"]
	return ok && forceReconcile == "true"
}

var resourcesChart = charts.MustGetChart("tenant-resources", nil)

func (r *CentralReconciler) checkSecretExists(
	ctx context.Context,
	remoteCentralNamespace string,
	secretName string,
) (bool, error) {
	secret := &corev1.Secret{}
	err := r.client.Get(ctx, ctrlClient.ObjectKey{Namespace: remoteCentralNamespace, Name: secretName}, secret)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			return false, nil
		}

		return false, fmt.Errorf("getting secret %s/%s: %w", remoteCentralNamespace, secretName, err)
	}

	return true, nil
}

func (r *CentralReconciler) ensureSecretExists(
	ctx context.Context,
	namespace string,
	secretName string,
	secretModifyFunc func(secret *corev1.Secret) error,
) error {
	secret := &corev1.Secret{}
	secretKey := ctrlClient.ObjectKey{Name: secretName, Namespace: namespace} // pragma: allowlist secret

	err := r.client.Get(ctx, secretKey, secret) // pragma: allowlist secret
	if err != nil && !apiErrors.IsNotFound(err) {
		return fmt.Errorf("getting %s/%s secret: %w", namespace, secretName, err)
	}
	if err == nil {
		modificationErr := secretModifyFunc(secret)
		if modificationErr != nil {
			return fmt.Errorf("updating %s/%s secret content: %w", namespace, secretName, modificationErr)
		}
		if updateErr := r.client.Update(ctx, secret); updateErr != nil { // pragma: allowlist secret
			return fmt.Errorf("updating %s/%s secret: %w", namespace, secretName, updateErr)
		}

		return nil
	}

	// Create secret if it does not exist.
	secret = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{ // pragma: allowlist secret
			Name:      secretName,
			Namespace: namespace,
			Labels:    map[string]string{k8s.ManagedByLabelKey: k8s.ManagedByFleetshardValue},
			Annotations: map[string]string{
				managedServicesAnnotation: "true",
			},
		},
	}

	if modificationErr := secretModifyFunc(secret); modificationErr != nil {
		return fmt.Errorf("initializing %s/%s secret payload: %w", namespace, secretName, modificationErr)
	}
	if createErr := r.client.Create(ctx, secret); createErr != nil { // pragma: allowlist secret
		return fmt.Errorf("creating %s/%s secret: %w", namespace, secretName, createErr)
	}
	return nil
}

func (r *CentralReconciler) configureAdditionalAuthProvider(secret *corev1.Secret, central private.ManagedCentral) error {
	if secret.Data == nil {
		secret.Data = make(map[string][]byte)
	}

	authProviderConfig := getAdditionalAuthProvider(central)

	rawAuthProviderBytes, err := yaml.Marshal(authProviderConfig)
	if err != nil {
		return fmt.Errorf("marshaling additional auth provider configuration: %w", err)
	}
	secret.Data[additionalAuthProviderConfigKey] = rawAuthProviderBytes
	return nil
}

func getAdditionalAuthProvider(central private.ManagedCentral) *declarativeconfig.AuthProvider {
	auth := central.Spec.AdditionalAuth
	// Assume that if name is not specified, no additional auth provider is configured.
	if auth.Name == "" {
		return nil
	}
	groups := make([]declarativeconfig.Group, 0, len(auth.Groups))
	for _, group := range auth.Groups {
		groups = append(groups, declarativeconfig.Group{
			AttributeKey:   group.Key,
			AttributeValue: group.Value,
			RoleName:       group.Role,
		})
	}

	requiredAttributes := make([]declarativeconfig.RequiredAttribute, 0, len(auth.RequiredAttributes))
	for _, requiredAttribute := range auth.RequiredAttributes {
		requiredAttributes = append(requiredAttributes, declarativeconfig.RequiredAttribute{
			AttributeKey:   requiredAttribute.Key,
			AttributeValue: requiredAttribute.Value,
		})
	}

	claimMappings := make([]declarativeconfig.ClaimMapping, 0, len(auth.ClaimMappings))
	for _, claimMapping := range auth.ClaimMappings {
		claimMappings = append(claimMappings, declarativeconfig.ClaimMapping{
			Path: claimMapping.Key,
			Name: claimMapping.Value,
		})
	}

	return &declarativeconfig.AuthProvider{
		Name:               auth.Name,
		UIEndpoint:         central.Spec.UiEndpoint.Host,
		ExtraUIEndpoints:   []string{"localhost:8443"},
		Groups:             groups,
		RequiredAttributes: requiredAttributes,
		ClaimMappings:      claimMappings,
		OIDCConfig: &declarativeconfig.OIDCConfig{
			Issuer:                    auth.Oidc.Issuer,
			CallbackMode:              auth.Oidc.CallbackMode,
			ClientID:                  auth.Oidc.ClientID,
			ClientSecret:              auth.Oidc.ClientSecret, // pragma: allowlist secret
			DisableOfflineAccessScope: auth.Oidc.DisableOfflineAccessScope,
		},
	}
}

// NewCentralReconciler ...
func NewCentralReconciler(k8sClient ctrlClient.Client, fleetmanagerClient *fleetmanager.Client, central private.ManagedCentral,
	managedDBProvisioningClient cloudprovider.DBClient, managedDBInitFunc postgres.CentralDBInitFunc,
	secretCipher cipher.Cipher, encryptionKeyGenerator cipher.KeyGenerator,
	opts CentralReconcilerOptions,
) *CentralReconciler {
	r := &CentralReconciler{
		client:                 k8sClient,
		fleetmanagerClient:     fleetmanagerClient,
		central:                central,
		status:                 pointer.Int32(FreeStatus),
		useRoutes:              opts.UseRoutes,
		wantsAuthProvider:      opts.WantsAuthProvider,
		routeService:           k8s.NewRouteService(k8sClient, &opts.RouteParameters),
		secretBackup:           k8s.NewSecretBackup(k8sClient, opts.ManagedDBEnabled),
		secretCipher:           secretCipher, // pragma: allowlist secret
		egressProxyImage:       opts.EgressProxyImage,
		telemetry:              opts.Telemetry,
		clusterName:            opts.ClusterName,
		environment:            opts.Environment,
		auditLogging:           opts.AuditLogging,
		encryptionKeyGenerator: encryptionKeyGenerator,

		managedDBEnabled:            opts.ManagedDBEnabled,
		managedDBProvisioningClient: managedDBProvisioningClient,
		managedDBInitFunc:           managedDBInitFunc,

		verifyAuthProviderFunc: hasAuthProvider,
		tenantImagePullSecret:  []byte(opts.TenantImagePullSecret),

		resourcesChart: resourcesChart,
		clock:          realClock{},
	}
	r.needsReconcileFunc = r.needsReconcile

	r.restoreCentralSecretsFunc = r.restoreCentralSecrets //pragma: allowlist secret
	r.areSecretsStoredFunc = r.areSecretsStored           //pragma: allowlist secret
	return r
}

func obscureSecrets(secrets map[string]string) map[string]string {
	obscuredSecrets := make(map[string]string, len(secrets))

	for key := range secrets {
		obscuredSecrets[key] = "secret-value"
	}

	return obscuredSecrets
}

type clock interface {
	Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time {
	return time.Now()
}

type fakeClock struct {
	NowTime time.Time
}

func (f fakeClock) Now() time.Time {
	return f.NowTime
}
