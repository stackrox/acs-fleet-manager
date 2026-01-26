// Package runtime ...
package runtime

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/stackrox/rox/operator/api/v1alpha1"
	"github.com/stackrox/rox/pkg/concurrency"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/stackrox/acs-fleet-manager/fleetshard/config"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/cloudprovider"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/cloudprovider/awsclient"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/postgres"
	centralReconciler "github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/reconciler"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/cipher"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/fleetshardmetrics"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/k8s"
	"github.com/stackrox/acs-fleet-manager/internal/central/pkg/api/private"
	fmAPI "github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
	fleetmanager "github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager/impl"
	"github.com/stackrox/acs-fleet-manager/pkg/features"
	"github.com/stackrox/acs-fleet-manager/pkg/logger"
)

// reconcilerRegistry contains a registry of a reconciler for each Central tenant. The key is the identifier of the
// Central instance.
// TODO(SimonBaeumer): set a unique identifier for the map key, currently the instance name is used
type reconcilerRegistry map[string]*centralReconciler.CentralReconciler

var reconciledCentralCountCache int32

var backoff = wait.Backoff{
	Duration: 1 * time.Second,
	Factor:   1.5,
	Jitter:   0.1,
	Steps:    15,
	Cap:      10 * time.Minute,
}

// Runtime represents the runtime to reconcile all centrals associated with the given cluster.
type Runtime struct {
	config                        *config.Config
	client                        *fmAPI.Client
	clusterID                     string
	reconcilers                   reconcilerRegistry
	k8sClient                     ctrlClient.Client
	dbProvisionClient             cloudprovider.DBClient
	statusResponseCh              chan private.DataPlaneCentralStatus
	secretCipher                  cipher.Cipher
	encryptionKeyGenerator        cipher.KeyGenerator
	runtimeApplicationsReconciler *runtimeApplicationsReconciler
}

// NewRuntime creates a new runtime
func NewRuntime(ctx context.Context, config *config.Config, k8sClient ctrlClient.Client) (*Runtime, error) {
	authOption := fleetmanager.Option{
		Static: fleetmanager.StaticOption{
			StaticToken: config.StaticToken,
		},
		ServiceAccount: fleetmanager.ServiceAccountOption{
			TokenFile: config.ServiceAccountTokenFile,
		},
	}
	auth, err := fleetmanager.NewAuth(ctx, config.AuthType, authOption)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create fleet manager authentication")
	}
	client, err := fleetmanager.NewClient(config.FleetManagerEndpoint, auth, fleetmanager.WithUserAgent(
		fmt.Sprintf("fleetshard-synchronizer/%s", config.ClusterID)),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create fleet manager client")
	}
	var dbProvisionClient cloudprovider.DBClient
	if config.ManagedDB.Enabled {
		dbProvisionClient, err = awsclient.NewRDSClient(config)
		if err != nil {
			return nil, fmt.Errorf("creating managed DB provisioning client: %v", err)
		}
	}

	secretCipher, err := cipher.NewCipher(config)
	if err != nil {
		return nil, fmt.Errorf("creating secretCipher: %w", err)
	}

	encryptionKeyGen, err := cipher.NewKeyGenerator(config)
	if err != nil {
		return nil, fmt.Errorf("creating encryption KeyGenerator: %w", err)
	}

	return &Runtime{
		config:                        config,
		k8sClient:                     k8sClient,
		client:                        client,
		clusterID:                     config.ClusterID,
		dbProvisionClient:             dbProvisionClient,
		reconcilers:                   make(reconcilerRegistry),
		secretCipher:                  secretCipher, // pragma: allowlist secret
		encryptionKeyGenerator:        encryptionKeyGen,
		runtimeApplicationsReconciler: newRuntimeApplicationsReconciler(k8sClient, config.ArgoCdNamespace),
	}, nil
}

// Stop stops the runtime
func (r *Runtime) Stop() {
}

// Start starts the fleetshard runtime and schedules
func (r *Runtime) Start() error {
	glog.Info("fleetshard runtime started")
	glog.Infof("Auth provider initialisation enabled: %v", r.config.CreateAuthProvider)

	routesAvailable := r.routesAvailable()

	argoReconcilerOpts := centralReconciler.ArgoReconcilerOptions{
		TenantDefaultArgoCdAppSourceTargetRevision: r.config.TenantDefaultArgoCdAppSourceTargetRevision,
		TenantDefaultArgoCdAppSourcePath:           r.config.TenantDefaultArgoCdAppSourcePath,
		TenantDefaultArgoCdAppSourceRepoURL:        r.config.TenantDefaultArgoCdAppSourceRepoURL,
		ArgoCdNamespace:                            r.config.ArgoCdNamespace,
		ManagedDBEnabled:                           r.config.ManagedDB.Enabled,
		ClusterName:                                r.config.ClusterName,
		Environment:                                r.config.Environment,
		WantsAuthProvider:                          r.config.CreateAuthProvider,
		Telemetry:                                  r.config.Telemetry,
	}

	reconcilerOpts := centralReconciler.CentralReconcilerOptions{
		UseRoutes:             routesAvailable,
		WantsAuthProvider:     r.config.CreateAuthProvider,
		ManagedDBEnabled:      r.config.ManagedDB.Enabled,
		ClusterName:           r.config.ClusterName,
		Environment:           r.config.Environment,
		AuditLogging:          r.config.AuditLogging,
		TenantImagePullSecret: r.config.TenantImagePullSecret, // pragma: allowlist secret
		ArgoReconcilerOptions: argoReconcilerOpts,
	}

	tenantCleanupOpts := centralReconciler.TenantCleanupOptions{
		ArgoReconcilerOptions: argoReconcilerOpts,
	}

	tenantCleanup := centralReconciler.NewTenantCleanup(
		r.k8sClient,
		tenantCleanupOpts,
	)

	ticker := concurrency.NewRetryTicker(func(ctx context.Context) (timeToNextTick time.Duration, err error) {
		list, _, err := r.client.PrivateAPI().GetCentrals(ctx, r.clusterID)
		if err != nil {
			err = errors.Wrapf(err, "retrieving list of managed centrals")
			glog.Error(err)
			return 0, err
		}

		if err := r.runtimeApplicationsReconciler.reconcile(ctx, list); err != nil {
			glog.Errorf("failed to reconcile runtime applications: %v", err)
		}

		// Start for each Central its own reconciler which can be triggered by sending a central to the receive channel.
		reconciledCentralCountCache = int32(len(list.Items))
		logger.InfoChangedInt32(&reconciledCentralCountCache, "Received central count changed: received %d centrals", reconciledCentralCountCache)
		reconcileResults := make(chan reconcileResult, len(list.Items))
		var wg sync.WaitGroup
		for _, central := range list.Items {
			if _, ok := r.reconcilers[central.Id]; !ok {
				r.reconcilers[central.Id] = centralReconciler.NewCentralReconciler(r.k8sClient, r.client,
					r.dbProvisionClient, postgres.InitializeDatabase, r.secretCipher, r.encryptionKeyGenerator, reconcilerOpts)
			}

			reconciler := r.reconcilers[central.Id]
			wg.Add(1)
			go func(reconciler *centralReconciler.CentralReconciler, central private.ManagedCentral) {
				defer wg.Done()
				fleetshardmetrics.MetricsInstance().IncActiveCentralReconcilations()
				defer fleetshardmetrics.MetricsInstance().DecActiveCentralReconcilations()

				// a 15 minutes timeout should cover the duration of a Reconcile call, including the provisioning of an RDS database
				ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
				defer cancel()

				status, err := reconciler.Reconcile(ctx, central)
				fleetshardmetrics.MetricsInstance().IncCentralReconcilations()
				submitReconcileResult(central, status, err, reconcileResults)
			}(reconciler, central)

			reconcilePaused, err := r.isReconcilePaused(ctx, central)
			if err != nil {
				glog.Warningf("Error getting pause annotation status: %v", err)
			} else {
				fleetshardmetrics.MetricsInstance().SetPauseReconcileStatus(central.Id, reconcilePaused)
			}
		}

		go func() {
			wg.Wait()
			close(reconcileResults)
			r.handleReconcileResults(reconcileResults)
		}()

		if reconcilerOpts.ManagedDBEnabled {
			accountQuotas, err := r.dbProvisionClient.GetAccountQuotas(ctx)
			if err != nil {
				glog.Warningf("Error retrieving account quotas: %v", err)
			} else {
				fleetshardmetrics.MetricsInstance().SetDatabaseAccountQuotas(accountQuotas)
			}
		}

		r.deleteStaleReconcilers(&list)

		if features.ClusterMigration.Enabled() {
			if err := tenantCleanup.DeleteStaleTenantK8sResources(ctx, &list); err != nil {
				glog.Errorf("Failed to delete stale tenant k8s resources: %s", err.Error())
			}
		}

		return r.config.RuntimePollPeriod, nil
	}, 10*time.Minute, backoff)

	err := ticker.Start()
	if err != nil {
		return fmt.Errorf("starting ticker: %w", err)
	}

	return nil
}

type reconcileResult struct {
	central private.ManagedCentral
	status  private.DataPlaneCentralStatus
	err     error
}

func submitReconcileResult(central private.ManagedCentral, status *private.DataPlaneCentralStatus, err error, results chan<- reconcileResult) {
	if err != nil {
		results <- reconcileResult{
			central: central,
			err:     err,
		}
	} else if status == nil {
		results <- reconcileResult{
			central: central,
			err:     centralReconciler.ErrCentralNotChanged,
		}
	} else {
		results <- reconcileResult{
			central: central,
			status:  *status,
		}
	}
}

func (r *Runtime) handleReconcileResults(results <-chan reconcileResult) {
	statuses := map[string]private.DataPlaneCentralStatus{}
	statusesCount := centralReconciler.StatusesCount{}
	defer statusesCount.SubmitMetric()

	for result := range results {
		central := result.central
		if err := result.err; err != nil {
			if centralReconciler.IsSkippable(err) {
				glog.V(10).Infof("Skip sending the status for central %s/%s: %v", central.Metadata.Namespace, central.Metadata.Name, err)
				statusesCount.IncrementRemote(central.RequestStatus) // get remote status
			} else {
				fleetshardmetrics.MetricsInstance().IncCentralReconcilationErrors()
				glog.Errorf("Unexpected error occurred %s/%s: %s", central.Metadata.Namespace, central.Metadata.Name, err.Error())
			}
		} else {
			statusesCount.IncrementCurrent(result.status)
			statuses[central.Id] = result.status
		}
	}
	if len(statuses) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()
	_, err := r.client.PrivateAPI().UpdateCentralClusterStatus(ctx, r.clusterID, statuses)
	if err != nil {
		glog.Errorf("updating statuses for Centrals: %v", err)
	}
}

func (r *Runtime) deleteStaleReconcilers(list *private.ManagedCentralList) {
	// This map collects all central ids in the current list, it is later used to find and delete all reconcilers of
	// centrals that are no longer in the GetManagedCentralList
	centralIds := map[string]struct{}{}
	for _, central := range list.Items {
		centralIds[central.Id] = struct{}{}
	}

	for key := range r.reconcilers {
		if _, hasKey := centralIds[key]; !hasKey {
			delete(r.reconcilers, key)
		}
	}
}

func (r *Runtime) routesAvailable() bool {
	available, err := k8s.IsRoutesResourceEnabled(r.k8sClient)
	if err != nil {
		glog.Errorf("Skip checking OpenShift routes availability due to an error: %v", err)
		return true // make an optimistic assumption that routes can be created despite the error
	}
	glog.Infof("OpenShift Routes available: %t", available)
	if !available {
		glog.Warning("Most likely the application is running on a plain Kubernetes cluster. " +
			"Such setup is unsupported and can be used for development only!")
		return false
	}
	return true
}

func (r *Runtime) isReconcilePaused(ctx context.Context, remoteCentral private.ManagedCentral) (bool, error) {
	central := &v1alpha1.Central{}
	err := r.k8sClient.Get(ctx, ctrlClient.ObjectKey{
		Namespace: remoteCentral.Metadata.Namespace,
		Name:      remoteCentral.Metadata.Name}, central)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			return false, nil
		}

		return false, errors.Wrapf(err, "getting CR for Central: %s", remoteCentral.Id)
	}

	if central.Annotations == nil {
		return false, nil
	}

	value, exists := central.Annotations[centralReconciler.PauseReconcileAnnotation]
	if !exists {
		return false, nil
	}

	if value == "true" {
		return true, nil
	}

	return false, nil
}
