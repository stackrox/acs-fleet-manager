// Package reconciler provides update, delete and create logic for managing Central instances.
package reconciler

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"sync/atomic"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/golang/glog"
	"github.com/hashicorp/go-multierror"
	openshiftRouteV1 "github.com/openshift/api/route/v1"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/fleetshard/config"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/cloudprovider"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/postgres"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/cipher"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/k8s"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/util"
	centralConstants "github.com/stackrox/acs-fleet-manager/internal/central/constants"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/private"
	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
	centralNotifierUtils "github.com/stackrox/rox/central/notifiers/utils"
	"github.com/stackrox/rox/pkg/declarativeconfig"
	"github.com/stackrox/rox/pkg/random"
	"golang.org/x/exp/maps"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/pointer"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

// FreeStatus ...
const (
	FreeStatus int32 = iota
	BlockedStatus

	PauseReconcileAnnotation = "stackrox.io/pause-reconcile"

	argoCdManagedBy = "argocd.argoproj.io/managed-by"

	managedServicesAnnotation = "platform.stackrox.io/managed-services"
	orgNameAnnotationKey      = "rhacs.redhat.com/org-name"

	ovnACLLoggingAnnotationKey     = "k8s.ovn.org/acl-logging"
	ovnACLLoggingAnnotationDefault = "{\"deny\": \"warning\"}"

	labelManagedByFleetshardValue = "rhacs-fleetshard"
	instanceLabelKey              = "app.kubernetes.io/instance"
	instanceTypeLabelKey          = "rhacs.redhat.com/instance-type"
	managedByLabelKey             = "app.kubernetes.io/managed-by"
	orgIDLabelKey                 = "rhacs.redhat.com/org-id"
	TenantIDLabelKey              = "rhacs.redhat.com/tenant"

	ProbeLabelKey = "rhacs.redhat.com/probe"

	centralExpiredAtKey = "rhacs.redhat.com/expired-at"

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

	authProviderDeclarativeConfigKey = "default-sso-auth-provider"
	additionalAuthProviderConfigKey  = "additional-auth-provider"

	tenantImagePullSecretName = "stackrox" // pragma: allowlist secret
)

type verifyAuthProviderExistsFunc func(ctx context.Context, central private.ManagedCentral, client ctrlClient.Client) (bool, error)
type needsReconcileFunc func(changed bool, central private.ManagedCentral, storedSecrets []string) bool
type restoreCentralSecretsFunc func(ctx context.Context, remoteCentral private.ManagedCentral) error
type areSecretsStoredFunc func(secretsStored []string) bool

type encryptedSecrets struct {
	secrets   map[string]string
	sha256Sum string
}

// CentralReconcilerOptions are the static options for creating a reconciler.
type CentralReconcilerOptions struct {
	UseRoutes             bool
	WantsAuthProvider     bool
	ManagedDBEnabled      bool
	ClusterName           string
	Environment           string
	AuditLogging          config.AuditLogging
	TenantImagePullSecret string
	ArgoReconcilerOptions ArgoReconcilerOptions
}

// CentralReconciler is a reconciler tied to a one Central instance. It installs, updates and deletes Central instances
// in its Reconcile function.
type CentralReconciler struct {
	client                 ctrlClient.Client
	fleetmanagerClient     *fleetmanager.Client
	status                 *int32
	lastCentralHash        [16]byte
	lastCentralHashTime    time.Time
	useRoutes              bool
	Resources              bool
	namespaceReconciler    *namespaceReconciler
	argoReconciler         *argoReconciler
	tenantCleanup          *TenantCleanup
	routeService           *k8s.RouteService
	secretBackup           *k8s.SecretBackup
	secretCipher           cipher.Cipher
	clusterName            string
	environment            string
	auditLogging           config.AuditLogging
	encryptionKeyGenerator cipher.KeyGenerator
	uiReachabilityChecker  CentralUIReachabilityChecker

	managedDbReconciler *managedDbReconciler
	managedDBEnabled    bool

	wantsAuthProvider     bool
	tenantImagePullSecret []byte
	clock                 clock

	areSecretsStoredFunc      areSecretsStoredFunc
	needsReconcileFunc        needsReconcileFunc
	restoreCentralSecretsFunc restoreCentralSecretsFunc
}

// Reconcile takes a private.ManagedCentral and tries to install it into the cluster managed by the fleet-shard.
// It tries to create a namespace for the Central and applies necessary updates to the resource.
// TODO(sbaumer): Check correct Central gets reconciled
// TODO(sbaumer): Should an initial ManagedCentral be added on reconciler creation?
func (r *CentralReconciler) Reconcile(ctx context.Context, remoteCentral private.ManagedCentral) (*private.DataPlaneCentralStatus, error) {
	remoteCentralNamespace := remoteCentral.Metadata.Namespace
	remoteCentralName := remoteCentral.Metadata.Name

	// Only allow to start reconcile function once
	if !atomic.CompareAndSwapInt32(r.status, FreeStatus, BlockedStatus) {
		return nil, ErrBusy
	}
	defer atomic.StoreInt32(r.status, FreeStatus)

	centralHash, err := r.computeCentralHash(remoteCentral)
	if err != nil {
		return nil, errors.Wrap(err, "computing central hash")
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

	needsReconcile := r.needsReconcileFunc(changed, remoteCentral, remoteCentral.Metadata.SecretsStored)

	if !needsReconcile && isRemoteCentralReady(&remoteCentral) {
		shouldUpdateCentralHash = true
		return nil, ErrCentralNotChanged
	}

	glog.Infof("Start reconcile central %s/%s", remoteCentralNamespace, remoteCentralName)

	if remoteCentral.Metadata.DeletionTimestamp != "" {
		status, err := r.reconcileInstanceDeletion(ctx, remoteCentral)
		shouldUpdateCentralHash = err == nil
		return status, err
	}

	ns := r.getDesiredNamespace(remoteCentral)
	if err := r.namespaceReconciler.reconcile(ctx, ns); err != nil {
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

	centralDBConnectionString := ""
	if r.managedDBEnabled {
		centralDBConnectionString, err = r.managedDbReconciler.getCentralDBConnectionString(ctx, remoteCentral)
		if err != nil {
			return nil, fmt.Errorf("getting Central DB connection string: %w", err)
		}
	}

	if err := r.argoReconciler.ensureApplicationExists(ctx, remoteCentral, centralDBConnectionString); err != nil {
		return nil, errors.Wrapf(err, "unable to install ArgoCD application for central %s/%s", remoteCentralNamespace, remoteCentralName)
	}

	if err = r.reconcileDeclarativeConfigurationData(ctx, remoteCentral); err != nil {
		return nil, err
	}

	// Check whether deployment is ready.
	centralDeploymentReady, err := isCentralDeploymentReady(ctx, r.client, remoteCentralNamespace)
	if err != nil {
		return nil, err
	}

	if err = r.ensureSecretHasOwnerReference(ctx, k8s.CentralTLSSecretName, remoteCentral); err != nil {
		return nil, err
	}

	if !centralDeploymentReady {
		if isRemoteCentralProvisioning(remoteCentral) && !needsReconcile { // no changes detected, wait until central become ready
			return nil, ErrCentralNotChanged
		}
		return installingStatus(), nil
	}

	if r.useRoutes && !isRemoteCentralReady(&remoteCentral) {
		// Check whether central UI host is reachable over HTTP.
		centralUIReachable, err := r.uiReachabilityChecker.IsCentralUIHostReachable(ctx, remoteCentral.Spec.UiHost)
		if err != nil {
			return nil, err
		}
		if !centralUIReachable {
			if isRemoteCentralProvisioning(remoteCentral) && !needsReconcile { // no changes detected, wait until central UI becomes reachable
				return nil, ErrCentralNotChanged
			}
			return installingStatus(), nil
		}
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

func (r *CentralReconciler) restoreCentralSecrets(ctx context.Context, remoteCentral private.ManagedCentral) error {
	restoreSecrets := []string{}
	for _, secretName := range remoteCentral.Metadata.SecretsStored { // pragma: allowlist secret
		exists, err := checkSecretExists(ctx, r.client, remoteCentral.Metadata.Namespace, secretName)
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

	glog.Info(fmt.Sprintf("Restore secret for tenant: %s/%s", remoteCentral.Id, remoteCentral.Metadata.Namespace), restoreSecrets)
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

func (r *CentralReconciler) reconcileInstanceDeletion(ctx context.Context, remoteCentral private.ManagedCentral) (*private.DataPlaneCentralStatus, error) {
	remoteCentralName := remoteCentral.Metadata.Name
	remoteCentralNamespace := remoteCentral.Metadata.Namespace

	deleted, err := r.ensureCentralDeleted(ctx, remoteCentral)
	if err != nil {
		return nil, errors.Wrapf(err, "delete central %s/%s", remoteCentralNamespace, remoteCentralName)
	}
	if deleted {
		return deletedStatus(), nil
	}
	return nil, ErrDeletionInProgress
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
		UIEndpoint:       remoteCentral.Spec.UiHost,
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
			{
				Path: "deprecated_sub",
				Name: "userid",
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
	return ensureSecretExists(
		ctx,
		r.client,
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

func (r *CentralReconciler) collectReconciliationStatus(ctx context.Context, remoteCentral *private.ManagedCentral) (*private.DataPlaneCentralStatus, error) {
	status := readyStatus()
	// Do not report routes statuses if:
	// 1. Routes are not used on the cluster
	// 2. Central request is in status "Ready" - assuming that routes are already reported and saved
	if r.useRoutes && !isRemoteCentralReady(remoteCentral) {
		var err error
		status.Routes, err = r.getRoutesStatuses(ctx, remoteCentral)
		if err != nil {
			return nil, err
		}
	}

	// Only report secrets if Central is ready, to ensure we're not trying to get secrets before they are created.
	if isRemoteCentralReady(remoteCentral) {
		encSecrets, err := r.collectSecretsEncrypted(ctx, remoteCentral)
		if err != nil {
			return nil, err
		}

		// Only report secrets if data hash differs to make sure we don't produce huge amount of data
		// if no update is required on the fleet-manager DB
		if encSecrets.sha256Sum != remoteCentral.Metadata.SecretDataSha256Sum { // pragma: allowlist secret
			status.Secrets = encSecrets.secrets               // pragma: allowlist secret
			status.SecretDataSha256Sum = encSecrets.sha256Sum // pragma: allowlist secret
		}
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

	for i := range secretsStoredSize {
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

func (r *CentralReconciler) collectSecretsEncrypted(ctx context.Context, remoteCentral *private.ManagedCentral) (encryptedSecrets, error) {
	secrets, err := r.collectSecrets(ctx, remoteCentral)
	if err != nil {
		return encryptedSecrets{}, err
	}

	encSecrets, err := r.encryptSecrets(secrets)
	if err != nil {
		return encSecrets, fmt.Errorf("encrypting secrets for namespace: %s: %w", remoteCentral.Metadata.Namespace, err)
	}

	return encSecrets, nil
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

// encryptSecrets return the encrypted secrets and a sha256 sum of secret data to check if secrets
// need update later on
func (r *CentralReconciler) encryptSecrets(secrets map[string]*corev1.Secret) (encryptedSecrets, error) {
	encSecrets := encryptedSecrets{secrets: map[string]string{}}

	allSecretData := []byte{}
	// sort to ensure the loop always executed in the same order
	// otherwise the sha sum can differ across multiple invocations
	keys := maps.Keys(secrets)
	sort.Strings(keys)
	for _, key := range keys { // pragma: allowlist secret
		secret := secrets[key]
		secretBytes, err := json.Marshal(secret)
		if err != nil {
			return encSecrets, fmt.Errorf("error marshaling secret for encryption: %s: %w", key, err)
		}

		// sort to ensure the loop always executed in the same order
		// otherwise the sha sum can differ across multiple invocations
		dataKeys := maps.Keys(secret.Data)
		sort.Strings(dataKeys)
		for _, dataKey := range dataKeys {
			allSecretData = append(allSecretData, secret.Data[dataKey]...)
		}

		encryptedBytes, err := r.secretCipher.Encrypt(secretBytes)
		if err != nil {
			return encSecrets, fmt.Errorf("encrypting secret: %s: %w", key, err)
		}

		encSecrets.secrets[key] = base64.StdEncoding.EncodeToString(encryptedBytes)
	}

	secretSum := sha256.Sum256(allSecretData)
	secretShaStr := base64.StdEncoding.EncodeToString(secretSum[:])
	encSecrets.sha256Sum = secretShaStr

	return encSecrets, nil
}

// ensureSecretHasOwnerReference is used to make sure the central-tls secret has it's
// owner reference properly set after a restore operation so that the automatic cert rotation
// in the operator is working
func (r *CentralReconciler) ensureSecretHasOwnerReference(ctx context.Context, secretName string, remoteCentral private.ManagedCentral) error {
	namespace := remoteCentral.Metadata.Namespace
	secret, err := r.getSecret(namespace, secretName)
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

	centralCRList := &unstructured.UnstructuredList{}
	centralCRList.SetGroupVersionKind(k8s.CentralGVK)

	if err := r.client.List(ctx, centralCRList, &ctrlClient.ListOptions{Namespace: namespace}); err != nil {
		return fmt.Errorf("getting current central CR from k8s: %w", err)
	}

	if len(centralCRList.Items) == 0 {
		return fmt.Errorf("no central CR found in namespaces: %q", namespace)
	}

	centralCR := centralCRList.Items[0]

	secret.OwnerReferences = []metav1.OwnerReference{
		*metav1.NewControllerRef(&centralCR, k8s.CentralGVK),
	}

	if err := r.client.Update(ctx, secret); err != nil {
		return fmt.Errorf("updating %s secret: %w", k8s.CentralTLSSecretName, err)
	}

	return nil
}

func isRemoteCentralProvisioning(remoteCentral private.ManagedCentral) bool {
	return remoteCentral.RequestStatus == centralConstants.CentralRequestStatusProvisioning.String()
}

func isRemoteCentralReady(remoteCentral *private.ManagedCentral) bool {
	return remoteCentral.RequestStatus == centralConstants.CentralRequestStatusReady.String()
}

// getRoutesStatuses returns the list of the routes statuses required for Central to be ready.
// Returns error if failed to find the routes ingresses or when AT LEAST ONE required route is unavailable.
// This is because after Central is considered ready, fleet manager stores the route information.
// Therefore, all required routes must be reported when Central is ready.
func (r *CentralReconciler) getRoutesStatuses(ctx context.Context, central *private.ManagedCentral) ([]private.DataPlaneCentralStatusRoutes, error) {
	unprocessedHosts := make(map[string]struct{}, 2)
	unprocessedHosts[central.Spec.UiHost] = struct{}{}
	unprocessedHosts[central.Spec.DataHost] = struct{}{}

	ingresses, err := r.routeService.FindAdmittedIngresses(ctx, central.Metadata.Namespace)
	if err != nil {
		return nil, fmt.Errorf("obtaining ingresses for routes statuses: %w", err)
	}
	var routesStatuses []private.DataPlaneCentralStatusRoutes
	for _, ingress := range ingresses {
		if _, exists := unprocessedHosts[ingress.Host]; exists {
			delete(unprocessedHosts, ingress.Host)
			routesStatuses = append(routesStatuses, getRouteStatus(ingress))
		}
	}
	if len(unprocessedHosts) != 0 {
		return nil, fmt.Errorf("unable to find admitted ingress")
	}
	return routesStatuses, nil
}

func getRouteStatus(ingress openshiftRouteV1.RouteIngress) private.DataPlaneCentralStatusRoutes {
	return private.DataPlaneCentralStatusRoutes{
		Domain: ingress.Host,
		Router: ingress.RouterCanonicalHostname,
	}
}

func (r *CentralReconciler) ensureCentralDeleted(ctx context.Context, remoteCentral private.ManagedCentral) (bool, error) {
	globalDeleted := true

	k8sResourcesDeleted, err := r.tenantCleanup.DeleteK8sResources(ctx, remoteCentral.Metadata.Namespace, remoteCentral.Metadata.Name)
	if err != nil {
		return false, err
	}
	globalDeleted = globalDeleted && k8sResourcesDeleted

	podsTerminated, err := r.ensureInstancePodsTerminated(ctx, remoteCentral)
	if err != nil {
		return false, err
	}
	globalDeleted = globalDeleted && podsTerminated

	if r.managedDBEnabled {
		dbDeleted, err := r.managedDbReconciler.ensureDeleted(ctx, remoteCentral)
		if err != nil {
			return false, err
		}
		globalDeleted = globalDeleted && dbDeleted
	}

	return globalDeleted, nil
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

func (r *CentralReconciler) getDesiredNamespace(c private.ManagedCentral) *corev1.Namespace {
	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:        c.Metadata.Namespace,
			Annotations: getNamespaceAnnotations(c),
			Labels:      r.getNamespaceLabels(c),
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

func (r *CentralReconciler) ensureEncryptionKeySecretExists(ctx context.Context, remoteCentralNamespace string) error {
	return ensureSecretExists(ctx, r.client, remoteCentralNamespace, centralEncryptionKeySecretName, r.populateEncryptionKeySecret)
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

func generateDBPassword() (string, error) {
	password, err := random.GenerateString(25, random.AlphanumericCharacters)
	if err != nil {
		return "", fmt.Errorf("generating DB password: %w", err)
	}

	return password, nil
}

func (r *CentralReconciler) ensureInstancePodsTerminated(ctx context.Context, remoteCentral private.ManagedCentral) (bool, error) {
	namespace := remoteCentral.Metadata.Namespace
	name := remoteCentral.Metadata.Name
	err := wait.PollUntilContextCancel(ctx, centralDeletePollInterval, true, func(ctx context.Context) (bool, error) {
		pods := &corev1.PodList{}
		labelKey := "app.kubernetes.io/part-of"
		labelValue := "stackrox-central-services"
		labels := map[string]string{labelKey: labelValue}
		err := r.client.List(ctx, pods,
			ctrlClient.InNamespace(namespace),
			ctrlClient.MatchingLabels(labels),
		)

		if err != nil {
			return false, fmt.Errorf("listing instance pods: %w", err)
		}

		// Make sure that the returned pods are central service pods in the correct namespace
		var filteredPods []corev1.Pod
		for _, pod := range pods.Items {
			if pod.Namespace != namespace {
				continue
			}
			if val, exists := pod.Labels[labelKey]; !exists || val != labelValue {
				continue
			}
			filteredPods = append(filteredPods, pod)
		}

		if len(filteredPods) == 0 {
			return true, nil
		}

		var podNames string
		for _, filteredPod := range filteredPods {
			podNames += filteredPod.Name + " "
		}

		glog.Infof("Waiting for pods to terminate: %s", podNames)
		return false, nil
	})

	if err != nil {
		return false, fmt.Errorf("waiting for pods to terminate: %w", err)
	}
	glog.Infof("All pods terminated for tenant %s in namespace %s.", name, namespace)
	return true, nil
}

func getTenantLabels(c private.ManagedCentral) map[string]string {
	labels := map[string]string{
		managedByLabelKey:    labelManagedByFleetshardValue,
		instanceLabelKey:     c.Metadata.Name,
		orgIDLabelKey:        c.Spec.Auth.OwnerOrgId,
		TenantIDLabelKey:     c.Id,
		instanceTypeLabelKey: c.Spec.InstanceType,
	}
	if c.Metadata.Internal {
		labels[ProbeLabelKey] = ""
	}
	return labels
}

func getTenantAnnotations(c private.ManagedCentral) map[string]string {
	return map[string]string{
		orgNameAnnotationKey: c.Spec.Auth.OwnerOrgName,
	}
}

func (r *CentralReconciler) getNamespaceLabels(c private.ManagedCentral) map[string]string {
	ret := map[string]string{}
	for k, v := range getTenantLabels(c) {
		ret[k] = v
	}
	ret[argoCdManagedBy] = r.argoReconciler.argoOpts.ArgoCdNamespace
	return ret
}

func getNamespaceAnnotations(c private.ManagedCentral) map[string]string {
	namespaceAnnotations := getTenantAnnotations(c)
	if c.Metadata.ExpiredAt != nil {
		namespaceAnnotations[centralExpiredAtKey] = c.Metadata.ExpiredAt.Format(time.RFC3339)
	}
	namespaceAnnotations[ovnACLLoggingAnnotationKey] = ovnACLLoggingAnnotationDefault
	return namespaceAnnotations
}

func (r *CentralReconciler) needsReconcile(changed bool, remoteCentral private.ManagedCentral, storedSecrets []string) bool {
	if !r.areSecretsStoredFunc(storedSecrets) {
		return true
	}

	if changed {
		return true
	}

	if r.clock.Now().Sub(r.lastCentralHashTime) > time.Minute*15 {
		return true
	}

	if force, ok := remoteCentral.Spec.TenantResourcesValues["forceReconcile"].(bool); ok && force {
		return true
	}

	return false
}

func (r *CentralReconciler) configureAdditionalAuthProvider(secret *corev1.Secret, central private.ManagedCentral) error {
	authProviderConfig := findAdditionalAuthProvider(central)
	if authProviderConfig == nil {
		return nil
	}
	if secret.Data == nil {
		secret.Data = make(map[string][]byte)
	}

	rawAuthProviderBytes, err := yaml.Marshal(authProviderConfig)
	if err != nil {
		return fmt.Errorf("marshaling additional auth provider configuration: %w", err)
	}
	secret.Data[additionalAuthProviderConfigKey] = rawAuthProviderBytes
	return nil
}

func findAdditionalAuthProvider(central private.ManagedCentral) *declarativeconfig.AuthProvider {
	authProvider := central.Spec.AdditionalAuthProvider
	// Assume that if name is not specified, no additional auth provider is configured.
	if authProvider.Name == "" {
		return nil
	}
	groups := make([]declarativeconfig.Group, 0, len(authProvider.Groups))
	for _, group := range authProvider.Groups {
		groups = append(groups, declarativeconfig.Group{
			AttributeKey:   group.Key,
			AttributeValue: group.Value,
			RoleName:       group.Role,
		})
	}

	requiredAttributes := make([]declarativeconfig.RequiredAttribute, 0, len(authProvider.RequiredAttributes))
	for _, requiredAttribute := range authProvider.RequiredAttributes {
		requiredAttributes = append(requiredAttributes, declarativeconfig.RequiredAttribute{
			AttributeKey:   requiredAttribute.Key,
			AttributeValue: requiredAttribute.Value,
		})
	}

	claimMappings := make([]declarativeconfig.ClaimMapping, 0, len(authProvider.ClaimMappings))
	for _, claimMapping := range authProvider.ClaimMappings {
		claimMappings = append(claimMappings, declarativeconfig.ClaimMapping{
			Path: claimMapping.Key,
			Name: claimMapping.Value,
		})
	}

	return &declarativeconfig.AuthProvider{
		Name:               authProvider.Name,
		UIEndpoint:         central.Spec.UiHost,
		ExtraUIEndpoints:   []string{"localhost:8443"},
		Groups:             groups,
		RequiredAttributes: requiredAttributes,
		ClaimMappings:      claimMappings,
		OIDCConfig: &declarativeconfig.OIDCConfig{
			Issuer:                    authProvider.Oidc.Issuer,
			CallbackMode:              authProvider.Oidc.CallbackMode,
			ClientID:                  authProvider.Oidc.ClientID,
			ClientSecret:              authProvider.Oidc.ClientSecret, // pragma: allowlist secret
			DisableOfflineAccessScope: authProvider.Oidc.DisableOfflineAccessScope,
		},
	}
}

// NewCentralReconciler ...
func NewCentralReconciler(k8sClient ctrlClient.Client, fleetmanagerClient *fleetmanager.Client,
	managedDBProvisioningClient cloudprovider.DBClient, managedDBInitFunc postgres.CentralDBInitFunc,
	secretCipher cipher.Cipher, encryptionKeyGenerator cipher.KeyGenerator,
	opts CentralReconcilerOptions,
) *CentralReconciler {
	nsReconciler := newNamespaceReconciler(k8sClient)
	argoReconciler := newArgoReconciler(k8sClient, opts.ArgoReconcilerOptions)
	dbReconciler := newManagedDbReconciler(k8sClient, managedDBProvisioningClient, managedDBInitFunc)
	tenantCleanupOptions := TenantCleanupOptions{ArgoReconcilerOptions: opts.ArgoReconcilerOptions}
	r := &CentralReconciler{
		client:                 k8sClient,
		fleetmanagerClient:     fleetmanagerClient,
		status:                 pointer.Int32(FreeStatus),
		useRoutes:              opts.UseRoutes,
		wantsAuthProvider:      opts.WantsAuthProvider,
		namespaceReconciler:    nsReconciler,
		argoReconciler:         argoReconciler,
		tenantCleanup:          NewTenantCleanup(k8sClient, tenantCleanupOptions),
		routeService:           k8s.NewRouteService(k8sClient),
		secretBackup:           k8s.NewSecretBackup(k8sClient, opts.ManagedDBEnabled),
		secretCipher:           secretCipher, // pragma: allowlist secret
		clusterName:            opts.ClusterName,
		environment:            opts.Environment,
		auditLogging:           opts.AuditLogging,
		encryptionKeyGenerator: encryptionKeyGenerator,
		uiReachabilityChecker:  NewHTTPCentralUIReachabilityChecker(),

		managedDbReconciler: dbReconciler,
		managedDBEnabled:    opts.ManagedDBEnabled,

		tenantImagePullSecret: []byte(opts.TenantImagePullSecret),

		clock: realClock{},
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
