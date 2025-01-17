package reconciler

import (
	"context"
	"fmt"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/cloudprovider"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/postgres"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	corev1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

type managedDbReconciler struct {
	client                      ctrlClient.Client
	managedDBProvisioningClient cloudprovider.DBClient
	managedDBInitFunc           postgres.CentralDBInitFunc
}

func newManagedDbReconciler(client ctrlClient.Client, managedDBProvisioningClient cloudprovider.DBClient, managedDBInitFunc postgres.CentralDBInitFunc) *managedDbReconciler {
	return &managedDbReconciler{
		client:                      client,
		managedDBProvisioningClient: managedDBProvisioningClient,
		managedDBInitFunc:           managedDBInitFunc,
	}
}

func (r *managedDbReconciler) ensureDeleted(ctx context.Context, remoteCentral private.ManagedCentral) (bool, error) {
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
	return true, nil
}

// getDatabaseID returns the cloud database ID for a central tenant.
// By default the database ID is equal to the centralID. It can be overridden by a ConfigMap.
func (r *managedDbReconciler) getDatabaseID(ctx context.Context, remoteCentralNamespace, centralID string) (string, error) {
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

func (r *managedDbReconciler) getCentralDBConnectionString(ctx context.Context, remoteCentral *private.ManagedCentral) (string, error) {
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

func (r *managedDbReconciler) centralDBUserExists(ctx context.Context, remoteCentralNamespace string) (bool, error) {
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

func (r *managedDbReconciler) ensureManagedCentralDBInitialized(ctx context.Context, remoteCentral *private.ManagedCentral) error {
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

func (r *managedDbReconciler) centralDBSecretExists(ctx context.Context, remoteCentralNamespace string) (bool, error) {
	return checkSecretExists(ctx, r.client, remoteCentralNamespace, centralDbSecretName)
}

func (r *managedDbReconciler) ensureCentralDBSecretExists(ctx context.Context, remoteCentralNamespace, userType, password string) error {
	setPasswordFunc := func(secret *corev1.Secret, userType, password string) {
		secret.Data = map[string][]byte{"password": []byte(password)}
		if secret.Annotations == nil {
			secret.Annotations = make(map[string]string)
		}
		secret.Annotations[dbUserTypeAnnotation] = userType
	}
	return ensureSecretExists(ctx, r.client, remoteCentralNamespace, centralDbSecretName, func(secret *corev1.Secret) error {
		setPasswordFunc(secret, userType, password)
		return nil
	})
}

func (r *managedDbReconciler) getDBPasswordFromSecret(ctx context.Context, centralNamespace string) (string, error) {
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
