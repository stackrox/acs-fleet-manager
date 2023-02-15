// Package reconciler provides update, delete and create logic for managing Central instances.
package reconciler

import (
	"bytes"
	"context"
	"fmt"
	"sync/atomic"

	"github.com/golang/glog"
	openshiftRouteV1 "github.com/openshift/api/route/v1"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/fleetshard/config"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/charts"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/cloudprovider"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/postgres"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/k8s"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/util"
	centralConstants "github.com/stackrox/acs-fleet-manager/internal/dinosaur/constants"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/converters"
	"github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"github.com/stackrox/rox/pkg/random"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	corev1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/utils/pointer"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

// FreeStatus ...
const (
	FreeStatus int32 = iota
	BlockedStatus

	helmReleaseName = "tenant-resources"

	managedServicesAnnotation = "platform.stackrox.io/managed-services"
	envAnnotationKey          = "rhacs.redhat.com/environment"
	clusterNameAnnotationKey  = "rhacs.redhat.com/cluster-name"
	orgNameAnnotationKey      = "rhacs.redhat.com/org-name"
	orgIDLabelKey             = "rhacs.redhat.com/org-id"
	tenantIDLabelKey          = "rhacs.redhat.com/tenant"

	dbUserTypeAnnotation = "platform.stackrox.io/user-type"
	dbUserTypeMaster     = "master"
	dbUserTypeCentral    = "central"
	dbCentralUserName    = "rhacs_central"

	centralDbSecretName = "central-db-password" // pragma: allowlist secret
)

// CentralReconcilerOptions are the static options for creating a reconciler.
type CentralReconcilerOptions struct {
	UseRoutes         bool
	WantsAuthProvider bool
	EgressProxyImage  string
	ManagedDBEnabled  bool
	Telemetry         config.Telemetry
	ClusterName       string
	Environment       string
}

// CentralReconciler is a reconciler tied to a one Central instance. It installs, updates and deletes Central instances
// in its Reconcile function.
type CentralReconciler struct {
	client            ctrlClient.Client
	central           private.ManagedCentral
	status            *int32
	lastCentralHash   [16]byte
	useRoutes         bool
	wantsAuthProvider bool
	hasAuthProvider   bool
	Resources         bool
	routeService      *k8s.RouteService
	egressProxyImage  string
	telemetry         config.Telemetry
	clusterName       string
	environment       string

	managedDBEnabled            bool
	managedDBProvisioningClient cloudprovider.DBClient
	managedDBInitFunc           postgres.CentralDBInitFunc

	resourcesChart *chart.Chart
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

	remoteCentralName := remoteCentral.Metadata.Name
	remoteCentralNamespace := remoteCentral.Metadata.Namespace
	if !changed && r.wantsAuthProvider == r.hasAuthProvider && isRemoteCentralReady(remoteCentral) {
		return nil, ErrCentralNotChanged
	}

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
	envVars := getProxyEnvVars(remoteCentralNamespace)

	central := &v1alpha1.Central{
		ObjectMeta: metav1.ObjectMeta{
			Name:      remoteCentralName,
			Namespace: remoteCentralNamespace,
			Labels: map[string]string{
				k8s.ManagedByLabelKey: k8s.ManagedByFleetshardValue,
				tenantIDLabelKey:      remoteCentral.Id,
			},
			Annotations: map[string]string{managedServicesAnnotation: "true"},
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
			},
			Customize: &v1alpha1.CustomizeSpec{
				EnvVars: envVars,
				Annotations: map[string]string{
					envAnnotationKey:         r.environment,
					clusterNameAnnotationKey: r.clusterName,
					orgNameAnnotationKey:     remoteCentral.Spec.Auth.OwnerOrgName,
				},
				Labels: map[string]string{
					orgIDLabelKey:    remoteCentral.Spec.Auth.OwnerOrgId,
					tenantIDLabelKey: remoteCentral.Id,
				},
			},
		},
	}

	// Check whether auth provider is actually created and this reconciler just is not aware of that.
	if r.wantsAuthProvider && !r.hasAuthProvider {
		exists, err := existsRHSSOAuthProvider(ctx, remoteCentral, r.client)
		if err != nil {
			return nil, err
		}
		// If sso.redhat.com auth provider exists, there is no need for admin/password login.
		// We also store whether auth provider exists within reconciler instance to avoid polluting network.
		if exists {
			glog.Infof("Auth provider for %s/%s already exists", remoteCentralNamespace, remoteCentralName)
			r.hasAuthProvider = true
		}
	}

	if r.hasAuthProvider {
		central.Spec.Central.AdminPasswordGenerationDisabled = pointer.Bool(true)
	}

	if remoteCentral.Metadata.DeletionTimestamp != "" {
		deleted, err := r.ensureCentralDeleted(ctx, remoteCentral, central)
		if err != nil {
			return nil, errors.Wrapf(err, "delete central %s/%s", remoteCentralNamespace, remoteCentralName)
		}
		if deleted {
			return deletedStatus(), nil
		}
		return nil, ErrDeletionInProgress
	}

	namespaceLabels := map[string]string{
		orgIDLabelKey:    remoteCentral.Spec.Auth.OwnerOrgId,
		tenantIDLabelKey: remoteCentral.Id,
	}
	if err := r.ensureNamespaceExists(remoteCentralNamespace, namespaceLabels); err != nil {
		return nil, errors.Wrapf(err, "unable to ensure that namespace %s exists", remoteCentralNamespace)
	}

	if err := r.ensureChartResourcesExist(ctx, remoteCentral); err != nil {
		return nil, errors.Wrapf(err, "unable to install chart resource for central %s/%s", central.GetNamespace(), central.GetName())
	}

	if r.managedDBEnabled {
		centralDBConnectionString, err := r.getCentralDBConnectionString(ctx, remoteCentral)
		if err != nil {
			return nil, fmt.Errorf("getting Central DB connection string: %w", err)
		}

		central.Spec.Central.DB = &v1alpha1.CentralDBSpec{
			IsEnabled:                v1alpha1.CentralDBEnabledPtr(v1alpha1.CentralDBEnabledTrue),
			ConnectionStringOverride: pointer.String(centralDBConnectionString),
			PasswordSecret: &v1alpha1.LocalSecretReference{
				Name: centralDbSecretName,
			},
		}
	}

	centralExists := true
	existingCentral := v1alpha1.Central{}
	err = r.client.Get(ctx, ctrlClient.ObjectKey{Namespace: remoteCentralNamespace, Name: remoteCentralName}, &existingCentral)
	if err != nil {
		if !apiErrors.IsNotFound(err) {
			return nil, errors.Wrapf(err, "unable to check the existence of central %s/%s", central.GetNamespace(), central.GetName())
		}
		centralExists = false
	}

	if !centralExists {
		if central.GetAnnotations() == nil {
			central.Annotations = map[string]string{}
		}
		if err := util.IncrementCentralRevision(central); err != nil {
			return nil, errors.Wrap(err, "incrementing central's revision")
		}

		glog.Infof("Creating central %s/%s", central.GetNamespace(), central.GetName())
		if err := r.client.Create(ctx, central); err != nil {
			return nil, errors.Wrapf(err, "creating new central %s/%s", remoteCentralNamespace, remoteCentralName)
		}
		glog.Infof("Central %s/%s created", central.GetNamespace(), central.GetName())
	} else {
		glog.Infof("Update central %s/%s", central.GetNamespace(), central.GetName())
		existingCentral.Spec = central.Spec

		if err := util.IncrementCentralRevision(&existingCentral); err != nil {
			return nil, errors.Wrap(err, "incrementing central's revision")
		}
		existingCentral.Spec = *central.Spec.DeepCopy()

		if err := r.client.Update(ctx, &existingCentral); err != nil {
			return nil, errors.Wrapf(err, "updating central %s/%s", central.GetNamespace(), central.GetName())
		}
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
	centralDeploymentReady, err := isCentralDeploymentReady(ctx, r.client, remoteCentral)
	if err != nil {
		return nil, err
	}
	if !centralDeploymentReady || !centralTLSSecretFound {
		if isRemoteCentralProvisioning(remoteCentral) && !changed { // no changes detected, wait until central become ready
			return nil, ErrCentralNotChanged
		}
		return installingStatus(), nil
	}

	// Skip auth provider initialisation if:
	// 1. Auth provider is already created
	// 2. OR reconciler creator specified auth provider not to be created
	// 3. OR Central request is in status "Ready" - meaning auth provider should've been initialised earlier
	if r.wantsAuthProvider && !r.hasAuthProvider && !isRemoteCentralReady(remoteCentral) {
		err = createRHSSOAuthProvider(ctx, remoteCentral, r.client)
		if err != nil {
			return nil, err
		}
		r.hasAuthProvider = true
	}

	status := readyStatus()
	// Do not report routes statuses if:
	// 1. Routes are not used on the cluster
	// 2. Central request is in status "Ready" - assuming that routes are already reported and saved
	if r.useRoutes && !isRemoteCentralReady(remoteCentral) {
		status.Routes, err = r.getRoutesStatuses(ctx, remoteCentralNamespace)
		if err != nil {
			return nil, err
		}
	}

	// Setting the last central hash must always be executed as the last step.
	// defer can't be used for this call because it is also executed after the reconcile failed.
	if err := r.setLastCentralHash(remoteCentral); err != nil {
		return nil, errors.Wrapf(err, "setting central reconcilation cache")
	}

	return status, nil
}

func isRemoteCentralProvisioning(remoteCentral private.ManagedCentral) bool {
	return remoteCentral.RequestStatus == centralConstants.CentralRequestStatusProvisioning.String()
}

func isRemoteCentralReady(remoteCentral private.ManagedCentral) bool {
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

func (r *CentralReconciler) ensureCentralDeleted(ctx context.Context, remoteCentral private.ManagedCentral, central *v1alpha1.Central) (bool, error) {
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

	if r.managedDBEnabled {
		err = r.managedDBProvisioningClient.EnsureDBDeprovisioned(remoteCentral.Id)
		if err != nil {
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

func (r *CentralReconciler) createTenantNamespace(ctx context.Context, namespace *corev1.Namespace, labels map[string]string) error {
	namespace.Labels = labels
	err := r.client.Create(ctx, namespace)
	if err != nil {
		return fmt.Errorf("creating namespace %q: %w", namespace.ObjectMeta.Name, err)
	}
	return nil
}

func (r *CentralReconciler) ensureNamespaceExists(name string, labels map[string]string) error {
	namespace, err := r.getNamespace(name)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			return r.createTenantNamespace(context.Background(), namespace, labels)
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

func (r *CentralReconciler) getCentralDBConnectionString(ctx context.Context, remoteCentral private.ManagedCentral) (string, error) {
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
		return "", fmt.Errorf("getting RDS DB connection data: %w", err)
	}
	return dbConnection.GetConnectionForUser(dbCentralUserName).AsConnectionString(), nil
}

func generateDBPassword() (string, error) {
	password, err := random.GenerateString(25, random.AlphanumericCharacters)
	if err != nil {
		return "", fmt.Errorf("generating DB password: %w", err)
	}

	return password, nil
}

func (r *CentralReconciler) ensureManagedCentralDBInitialized(ctx context.Context, remoteCentral private.ManagedCentral) error {
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

	err = r.managedDBProvisioningClient.EnsureDBProvisioned(ctx, remoteCentral.Id, dbMasterPassword)
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
	err = r.managedDBInitFunc(ctx, dbConnection.WithPassword(dbMasterPassword), dbCentralUserName, dbCentralPassword)
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
	secret := &corev1.Secret{}
	setPasswordFunc := func(secret *corev1.Secret, userType, password string) {
		secret.Data = map[string][]byte{"password": []byte(password)}
		if secret.Annotations == nil {
			secret.Annotations = make(map[string]string)
		}
		secret.Annotations[dbUserTypeAnnotation] = userType
	}

	err := r.client.Get(ctx, ctrlClient.ObjectKey{Namespace: remoteCentralNamespace, Name: centralDbSecretName}, secret)
	if err == nil {
		setPasswordFunc(secret, userType, password)
		err = r.client.Update(ctx, secret)
		if err != nil {
			return fmt.Errorf("updating Central DB secret: %w", err)
		}

		return nil
	}

	if !apiErrors.IsNotFound(err) {
		return fmt.Errorf("getting Central DB secret: %w", err)
	}

	// create secret if it does not exist
	secret = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      centralDbSecretName,
			Namespace: remoteCentralNamespace,
			Labels:    map[string]string{k8s.ManagedByLabelKey: k8s.ManagedByFleetshardValue},
			Annotations: map[string]string{
				managedServicesAnnotation: "true",
			},
		},
	}

	setPasswordFunc(secret, userType, password)
	err = r.client.Create(ctx, secret)
	if err != nil {
		return fmt.Errorf("creating Central DB secret: %w", err)
	}
	return nil
}

func (r *CentralReconciler) centralDBSecretExists(ctx context.Context, remoteCentralNamespace string) (bool, error) {
	secret := &corev1.Secret{}
	err := r.client.Get(ctx, ctrlClient.ObjectKey{Namespace: remoteCentralNamespace, Name: centralDbSecretName}, secret)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			return false, nil
		}

		return false, fmt.Errorf("getting central DB secret: %w", err)
	}

	return true, nil
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
	err := r.client.Get(ctx, ctrlClient.ObjectKey{Namespace: central.GetNamespace(), Name: central.GetName()}, central)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			return true, nil
		}

		return false, errors.Wrapf(err, "delete central CR %s/%s", central.GetNamespace(), central.GetName())
	}
	if err := r.client.Delete(ctx, central); err != nil {
		return false, errors.Wrapf(err, "delete central CR %s/%s", central.GetNamespace(), central.GetName())
	}
	glog.Infof("Central CR %s/%s is marked for deletion", central.GetNamespace(), central.GetName())
	return false, nil
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
		key := ctrlClient.ObjectKey{Namespace: obj.GetNamespace(), Name: obj.GetName()}
		var out unstructured.Unstructured
		out.SetGroupVersionKind(obj.GroupVersionKind())
		err := r.client.Get(ctx, key, &out)
		if err == nil {
			glog.V(10).Infof("Updating object %s/%s", obj.GetNamespace(), obj.GetName())
			obj.SetResourceVersion(out.GetResourceVersion())
			err := r.client.Update(ctx, obj)
			if err != nil {
				return fmt.Errorf("failed to update object %s/%s of type %v: %w", key.Namespace, key.Namespace, obj.GroupVersionKind(), err)
			}

			continue
		}
		if !apiErrors.IsNotFound(err) {
			return fmt.Errorf("failed to retrieve object %s/%s of type %v: %w", key.Namespace, key.Name, obj.GroupVersionKind(), err)
		}
		err = r.client.Create(ctx, obj)
		glog.V(10).Infof("Creating object %s/%s", obj.GetNamespace(), obj.GetName())
		if err != nil && !apiErrors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create object %s/%s of type %v: %w", key.Namespace, key.Name, obj.GroupVersionKind(), err)
		}
	}

	return nil
}

func (r *CentralReconciler) ensureChartResourcesDeleted(ctx context.Context, remoteCentral private.ManagedCentral) (bool, error) {
	vals, err := r.chartValues(remoteCentral)
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
	_, err := r.routeService.FindReencryptRoute(ctx, namespace)
	if err != nil && !apiErrors.IsNotFound(err) {
		return fmt.Errorf("retrieving reencrypt route for namespace %q: %w", namespace, err)
	}

	if apiErrors.IsNotFound(err) {
		err = r.routeService.CreateReencryptRoute(ctx, remoteCentral)
		if err != nil {
			return fmt.Errorf("creating reencrypt route for central %s: %w", remoteCentral.Id, err)
		}
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
	_, err := r.routeService.FindPassthroughRoute(ctx, namespace)
	if err != nil && !apiErrors.IsNotFound(err) {
		return fmt.Errorf("retrieving passthrough route for namespace %q: %w", namespace, err)
	}

	if apiErrors.IsNotFound(err) {
		err = r.routeService.CreatePassthroughRoute(ctx, remoteCentral)
		if err != nil {
			return fmt.Errorf("creating passthrough route for central %s: %w", remoteCentral.Id, err)
		}
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

func (r *CentralReconciler) chartValues(remoteCentral private.ManagedCentral) (chartutil.Values, error) {
	vals := chartutil.Values{
		"labels": map[string]interface{}{
			k8s.ManagedByLabelKey: k8s.ManagedByFleetshardValue,
		},
	}
	if r.egressProxyImage != "" {
		override := chartutil.Values{
			"egressProxy": chartutil.Values{
				"image": r.egressProxyImage,
			},
		}
		vals = chartutil.CoalesceTables(vals, override)
	}

	return vals, nil
}

var resourcesChart = charts.MustGetChart("tenant-resources")

// NewCentralReconciler ...
func NewCentralReconciler(k8sClient ctrlClient.Client, central private.ManagedCentral,
	managedDBProvisioningClient cloudprovider.DBClient, managedDBInitFunc postgres.CentralDBInitFunc,
	opts CentralReconcilerOptions,
) *CentralReconciler {
	return &CentralReconciler{
		client:            k8sClient,
		central:           central,
		status:            pointer.Int32(FreeStatus),
		useRoutes:         opts.UseRoutes,
		wantsAuthProvider: opts.WantsAuthProvider,
		routeService:      k8s.NewRouteService(k8sClient),
		egressProxyImage:  opts.EgressProxyImage,
		telemetry:         opts.Telemetry,
		clusterName:       opts.ClusterName,
		environment:       opts.Environment,

		managedDBEnabled:            opts.ManagedDBEnabled,
		managedDBProvisioningClient: managedDBProvisioningClient,
		managedDBInitFunc:           managedDBInitFunc,

		resourcesChart: resourcesChart,
	}
}
