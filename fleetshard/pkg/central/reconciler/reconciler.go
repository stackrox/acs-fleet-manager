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
	"strconv"
	"sync/atomic"
	"time"

	containerImage "github.com/containers/image/docker/reference"
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
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/converters"
	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
	"github.com/stackrox/acs-fleet-manager/pkg/features"
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
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/pointer"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

// FreeStatus ...
const (
	FreeStatus int32 = iota
	BlockedStatus

	PauseReconcileAnnotation  = "stackrox.io/pause-reconcile"
	ReconcileOperatorSelector = "rhacs.redhat.com/version-selector"

	helmReleaseName = "tenant-resources"

	centralPVCAnnotationKey   = "platform.stackrox.io/obsolete-central-pvc"
	managedServicesAnnotation = "platform.stackrox.io/managed-services"
	envAnnotationKey          = "rhacs.redhat.com/environment"
	clusterNameAnnotationKey  = "rhacs.redhat.com/cluster-name"
	orgNameAnnotationKey      = "rhacs.redhat.com/org-name"
	instanceTypeLabelKey      = "rhacs.redhat.com/instance-type"
	orgIDLabelKey             = "rhacs.redhat.com/org-id"
	tenantIDLabelKey          = "rhacs.redhat.com/tenant"

	auditLogNotifierKey  = "com.redhat.rhacs.auditLogNotifier"
	auditLogNotifierName = "Platform Audit Logs"
	auditLogTenantIDKey  = "tenant_id"

	dbUserTypeAnnotation = "platform.stackrox.io/user-type"
	dbUserTypeMaster     = "master"
	dbUserTypeCentral    = "central"
	dbCentralUserName    = "rhacs_central"

	centralDbSecretName       = "central-db-password" // pragma: allowlist secret
	centralDeletePollInterval = 5 * time.Second

	sensibleDeclarativeConfigSecretName = "cloud-service-sensible-declarative-configs" // pragma: allowlist secret
	manualDeclarativeConfigSecretName   = "cloud-service-manual-declarative-configs"   // pragma: allowlist secret

	authProviderDeclarativeConfigKey = "default-sso-auth-provider"
)

type verifyAuthProviderExistsFunc func(ctx context.Context, central private.ManagedCentral,
	client ctrlClient.Client) (bool, error)

// CentralReconcilerOptions are the static options for creating a reconciler.
type CentralReconcilerOptions struct {
	UseRoutes         bool
	WantsAuthProvider bool
	EgressProxyImage  string
	ManagedDBEnabled  bool
	Telemetry         config.Telemetry
	ClusterName       string
	Environment       string
	AuditLogging      config.AuditLogging
}

// CentralReconciler is a reconciler tied to a one Central instance. It installs, updates and deletes Central instances
// in its Reconcile function.
type CentralReconciler struct {
	client             ctrlClient.Client
	fleetmanagerClient *fleetmanager.Client
	central            private.ManagedCentral
	status             *int32
	lastCentralHash    [16]byte
	useRoutes          bool
	Resources          bool
	routeService       *k8s.RouteService
	secretBackup       *k8s.SecretBackup
	secretCipher       cipher.Cipher
	egressProxyImage   string
	telemetry          config.Telemetry
	clusterName        string
	environment        string
	auditLogging       config.AuditLogging

	managedDBEnabled            bool
	managedDBProvisioningClient cloudprovider.DBClient
	managedDBInitFunc           postgres.CentralDBInitFunc

	resourcesChart *chart.Chart

	wantsAuthProvider      bool
	hasAuthProvider        bool
	verifyAuthProviderFunc verifyAuthProviderExistsFunc
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

	changed, err := r.centralChanged(remoteCentral)
	if err != nil {
		return nil, errors.Wrapf(err, "checking if central changed")
	}
	needsReconcile := r.needsReconcile(changed, remoteCentral.ForceReconcile)

	if !needsReconcile && r.shouldSkipReadyCentral(remoteCentral) {
		return nil, ErrCentralNotChanged
	}

	glog.Infof("Start reconcile central %s/%s", remoteCentral.Metadata.Namespace, remoteCentral.Metadata.Name)

	remoteCentralNamespace := remoteCentral.Metadata.Namespace

	central, err := r.getInstanceConfig(&remoteCentral)
	if err != nil {
		return nil, err
	}

	if remoteCentral.Metadata.DeletionTimestamp != "" {
		return r.reconcileInstanceDeletion(ctx, &remoteCentral, central)
	}

	namespaceLabels := map[string]string{
		orgIDLabelKey:    remoteCentral.Spec.Auth.OwnerOrgId,
		tenantIDLabelKey: remoteCentral.Id,
	}
	namespaceAnnotations := map[string]string{
		orgNameAnnotationKey: remoteCentral.Spec.Auth.OwnerOrgName,
	}
	if err := r.ensureNamespaceExists(remoteCentralNamespace, namespaceLabels, namespaceAnnotations); err != nil {
		return nil, errors.Wrapf(err, "unable to ensure that namespace %s exists", remoteCentralNamespace)
	}

	err = r.restoreCentralSecrets(ctx, remoteCentral)
	if err != nil {
		return nil, err
	}

	if err := r.ensureChartResourcesExist(ctx, remoteCentral); err != nil {
		return nil, errors.Wrapf(err, "unable to install chart resource for central %s/%s", central.GetNamespace(), central.GetName())
	}

	if r.managedDBEnabled {
		if err = r.reconcileCentralDBConfig(ctx, &remoteCentral, central); err != nil {
			return nil, err
		}
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
				return nil, errors.Wrapf(err, "updating routes")
			}
		}
	}

	// Check whether deployment is ready.
	centralDeploymentReady, err := isCentralDeploymentReady(ctx, r.client, remoteCentral.Metadata.Namespace)
	if err != nil {
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

	// Setting the last central hash must always be executed as the last step.
	// defer can't be used for this call because it is also executed after the reconcile failed.
	if err := r.setLastCentralHash(remoteCentral); err != nil {
		return nil, errors.Wrapf(err, "setting central reconcilation cache")
	}

	glog.Infof("Returning central status %+v", status)

	return status, nil
}

func (r *CentralReconciler) getInstanceConfig(remoteCentral *private.ManagedCentral) (*v1alpha1.Central, error) {
	if remoteCentral == nil {
		return nil, errInvalidArguments
	}

	remoteCentralName := remoteCentral.Metadata.Name
	remoteCentralNamespace := remoteCentral.Metadata.Namespace

	monitoringExposeEndpointEnabled := v1alpha1.ExposeEndpointEnabled
	// Telemetry will only be enabled if the storage key is set _and_ the central is not an "internal" central created
	// from internal clients such as probe service or others.
	telemetryEnabled := r.telemetry.StorageKey != "" && !remoteCentral.Metadata.Internal

	centralResources, err := converters.ConvertPrivateResourceRequirementsToCoreV1(&remoteCentral.Spec.Central.Resources)
	if err != nil {
		return nil, errors.Wrap(err, "converting Central resources")
	}
	scannerAnalyzerResources, err := converters.ConvertPrivateResourceRequirementsToCoreV1(&remoteCentral.Spec.Scanner.Analyzer.Resources)
	if err != nil {
		return nil, errors.Wrap(err, "converting Scanner Analyzer resources")
	}
	scannerAnalyzerScaling := converters.ConvertPrivateScalingToV1(&remoteCentral.Spec.Scanner.Analyzer.Scaling)
	scannerDbResources, err := converters.ConvertPrivateResourceRequirementsToCoreV1(&remoteCentral.Spec.Scanner.Db.Resources)
	if err != nil {
		return nil, errors.Wrap(err, "converting Scanner DB resources")
	}

	// Set proxy configuration
	auditLoggingURL := url.URL{
		Host: r.auditLogging.Endpoint(false),
	}
	kubernetesURL := url.URL{
		Host: "kubernetes.default.svc.cluster.local.:443",
	}
	envVars := getProxyEnvVars(remoteCentralNamespace, auditLoggingURL, kubernetesURL)

	scannerComponentEnabled := v1alpha1.ScannerComponentEnabled

	central := &v1alpha1.Central{
		ObjectMeta: metav1.ObjectMeta{
			Name:      remoteCentralName,
			Namespace: remoteCentralNamespace,
			Labels: map[string]string{
				k8s.ManagedByLabelKey: k8s.ManagedByFleetshardValue,
				tenantIDLabelKey:      remoteCentral.Id,
				instanceTypeLabelKey:  remoteCentral.Spec.Central.InstanceType,
				orgIDLabelKey:         remoteCentral.Spec.Auth.OwnerOrgId,
			},
			Annotations: map[string]string{
				centralPVCAnnotationKey:   strconv.FormatBool(r.managedDBEnabled),
				managedServicesAnnotation: "true",
				orgNameAnnotationKey:      remoteCentral.Spec.Auth.OwnerOrgName,
			},
		},
		Spec: v1alpha1.CentralSpec{
			Central: &v1alpha1.CentralComponentSpec{
				Exposure: &v1alpha1.Exposure{
					Route: &v1alpha1.ExposureRoute{
						Enabled: pointer.Bool(r.useRoutes),
					},
				},
				Monitoring: &v1alpha1.Monitoring{
					ExposeEndpoint: &monitoringExposeEndpointEnabled,
				},
				DeploymentSpec: v1alpha1.DeploymentSpec{
					Resources: &centralResources,
				},
				Telemetry: &v1alpha1.Telemetry{
					Enabled: pointer.Bool(telemetryEnabled),
					Storage: &v1alpha1.TelemetryStorage{
						Endpoint: &r.telemetry.StorageEndpoint,
						Key:      &r.telemetry.StorageKey,
					},
				},
				DeclarativeConfiguration: &v1alpha1.DeclarativeConfiguration{
					Secrets: []v1alpha1.LocalSecretReference{
						{
							Name: sensibleDeclarativeConfigSecretName,
						},
						{
							Name: manualDeclarativeConfigSecretName,
						},
					},
				},
			},
			Scanner: &v1alpha1.ScannerComponentSpec{
				Analyzer: &v1alpha1.ScannerAnalyzerComponent{
					DeploymentSpec: v1alpha1.DeploymentSpec{
						Resources: &scannerAnalyzerResources,
					},
					Scaling: &scannerAnalyzerScaling,
				},
				DB: &v1alpha1.DeploymentSpec{
					Resources: &scannerDbResources,
				},
				Monitoring: &v1alpha1.Monitoring{
					ExposeEndpoint: &monitoringExposeEndpointEnabled,
				},
				ScannerComponent: &scannerComponentEnabled,
			},
			Customize: &v1alpha1.CustomizeSpec{
				EnvVars: envVars,
				Annotations: map[string]string{
					envAnnotationKey:         r.environment,
					clusterNameAnnotationKey: r.clusterName,
					orgNameAnnotationKey:     remoteCentral.Spec.Auth.OwnerOrgName,
				},
				Labels: map[string]string{
					orgIDLabelKey:        remoteCentral.Spec.Auth.OwnerOrgId,
					tenantIDLabelKey:     remoteCentral.Id,
					instanceTypeLabelKey: remoteCentral.Spec.Central.InstanceType,
				},
			},
		},
	}

	if features.TargetedOperatorUpgrades.Enabled() {
		// TODO: use GitRef as a LabelSelector
		image, err := containerImage.Parse(remoteCentral.Spec.OperatorImage)
		if err != nil {
			return nil, errors.Wrapf(err, "failed parse labelSelector")
		}
		var labelSelector string
		if tagged, ok := image.(containerImage.Tagged); ok {
			labelSelector = tagged.Tag()
		}
		errs := validation.IsValidLabelValue(labelSelector)
		if errs != nil {
			return nil, errors.Wrapf(err, "invalid labelSelector %s: %v", labelSelector, errs)
		}
		central.Labels[ReconcileOperatorSelector] = labelSelector

	}

	return central, nil
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

	centralDBConnectionString, err := r.getCentralDBConnectionString(ctx, remoteCentral)
	if err != nil {
		return fmt.Errorf("getting Central DB connection string: %w", err)
	}

	central.Spec.Central.DB = &v1alpha1.CentralDBSpec{
		IsEnabled:                v1alpha1.CentralDBEnabledPtr(v1alpha1.CentralDBEnabledTrue),
		ConnectionStringOverride: pointer.String(centralDBConnectionString),
		PasswordSecret: &v1alpha1.LocalSecretReference{
			Name: centralDbSecretName,
		},
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
	return &declarativeconfig.AuthProvider{
		Name:             authProviderName(remoteCentral),
		UIEndpoint:       remoteCentral.Spec.UiEndpoint.Host,
		ExtraUIEndpoints: []string{"localhost:8443"},
		Groups: []declarativeconfig.Group{
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
		},
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

		if err := r.client.Update(ctx, desiredCentral); err != nil {
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
	// Only report secrets once. Ensures we don't overwrite initial secrets with corrupted secrets
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
	expectedSecrets := k8s.GetWatchedSecrets()
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

		err = r.managedDBProvisioningClient.EnsureDBDeprovisioned(remoteCentral.Id, skipSnapshot)
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

// centralChanged compares the given central to the last central reconciled using a hash
func (r *CentralReconciler) centralChanged(central private.ManagedCentral) (bool, error) {
	currentHash, err := util.MD5SumFromJSONStruct(&central)
	if err != nil {
		return true, errors.Wrap(err, "hashing central")
	}

	return !bytes.Equal(r.lastCentralHash[:], currentHash[:]), nil
}

func (r *CentralReconciler) setLastCentralHash(central private.ManagedCentral) error {
	hash, err := util.MD5SumFromJSONStruct(&central)
	if err != nil {
		return fmt.Errorf("calculating MD5 from JSON: %w", err)
	}

	r.lastCentralHash = hash
	return nil
}

func (r *CentralReconciler) getNamespace(name string) (*corev1.Namespace, error) {
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	err := r.client.Get(context.Background(), ctrlClient.ObjectKey{Name: name}, namespace)
	if err != nil {
		// Propagate corev1.Namespace to the caller so that the namespace can be easily created
		return namespace, fmt.Errorf("retrieving resource for namespace %q from Kubernetes: %w", name, err)
	}
	return namespace, nil
}

func (r *CentralReconciler) createTenantNamespace(ctx context.Context, namespace *corev1.Namespace) error {
	err := r.client.Create(ctx, namespace)
	if err != nil {
		return fmt.Errorf("creating namespace %q: %w", namespace.ObjectMeta.Name, err)
	}
	return nil
}

func (r *CentralReconciler) ensureNamespaceExists(name string, labels map[string]string, annotations map[string]string) error {
	namespace, err := r.getNamespace(name)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			namespace.Annotations = annotations
			namespace.Labels = labels
			return r.createTenantNamespace(context.Background(), namespace)
		}
		return fmt.Errorf("getting namespace %s: %w", name, err)
	}
	return nil
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

	dbConnection, err := r.managedDBProvisioningClient.GetDBConnection(remoteCentral.Id)
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

	err = r.managedDBProvisioningClient.EnsureDBProvisioned(ctx, remoteCentral.Id, remoteCentral.Id, dbMasterPassword, remoteCentral.Metadata.Internal)
	if err != nil {
		return fmt.Errorf("provisioning RDS DB: %w", err)
	}

	dbConnection, err := r.managedDBProvisioningClient.GetDBConnection(remoteCentral.Id)
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
		if err := r.disablePauseReconcileIfPresent(ctx, central); err != nil {
			return false, err
		}

		if centralToDelete.GetDeletionTimestamp() == nil {
			glog.Infof("Marking Central CR %v for deletion", centralKey)
			if err := r.client.Delete(ctx, central); err != nil {
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

func (r *CentralReconciler) chartValues(_ private.ManagedCentral) (chartutil.Values, error) {
	if r.resourcesChart == nil {
		return nil, errors.New("resources chart is not set")
	}
	src := r.resourcesChart.Values
	dst := map[string]interface{}{
		"labels": map[string]interface{}{
			k8s.ManagedByLabelKey: k8s.ManagedByFleetshardValue,
		},
	}
	if r.egressProxyImage != "" {
		dst["egressProxy"] = map[string]interface{}{
			"image": r.egressProxyImage,
		}
	}
	return chartutil.CoalesceTables(dst, src), nil
}

func (r *CentralReconciler) shouldSkipReadyCentral(remoteCentral private.ManagedCentral) bool {
	return r.wantsAuthProvider == r.hasAuthProvider &&
		isRemoteCentralReady(&remoteCentral)
}

func (r *CentralReconciler) needsReconcile(changed bool, forceReconcile string) bool {
	return changed || forceReconcile == "always"
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

// NewCentralReconciler ...
func NewCentralReconciler(k8sClient ctrlClient.Client, fleetmanagerClient *fleetmanager.Client, central private.ManagedCentral,
	managedDBProvisioningClient cloudprovider.DBClient, managedDBInitFunc postgres.CentralDBInitFunc,
	secretCipher cipher.Cipher,
	opts CentralReconcilerOptions,
) *CentralReconciler {
	return &CentralReconciler{
		client:             k8sClient,
		fleetmanagerClient: fleetmanagerClient,
		central:            central,
		status:             pointer.Int32(FreeStatus),
		useRoutes:          opts.UseRoutes,
		wantsAuthProvider:  opts.WantsAuthProvider,
		routeService:       k8s.NewRouteService(k8sClient),
		secretBackup:       k8s.NewSecretBackup(k8sClient),
		secretCipher:       secretCipher, // pragma: allowlist secret
		egressProxyImage:   opts.EgressProxyImage,
		telemetry:          opts.Telemetry,
		clusterName:        opts.ClusterName,
		environment:        opts.Environment,
		auditLogging:       opts.AuditLogging,

		managedDBEnabled:            opts.ManagedDBEnabled,
		managedDBProvisioningClient: managedDBProvisioningClient,
		managedDBInitFunc:           managedDBInitFunc,

		verifyAuthProviderFunc: hasAuthProvider,

		resourcesChart: resourcesChart,
	}
}
