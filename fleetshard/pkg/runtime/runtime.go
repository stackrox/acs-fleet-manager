// Package runtime ...
package runtime

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/fleetshard/config"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/cloudprovider"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/cloudprovider/awsclient"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/operator"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/postgres"
	centralReconciler "github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/reconciler"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/cipher"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/fleetshardmetrics"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/k8s"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
	"github.com/stackrox/acs-fleet-manager/pkg/features"
	"github.com/stackrox/acs-fleet-manager/pkg/logger"
	"github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"github.com/stackrox/rox/pkg/concurrency"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

// reconcilerRegistry contains a registry of a reconciler for each Central tenant. The key is the identifier of the
// Central instance.
// TODO(SimonBaeumer): set a unique identifier for the map key, currently the instance name is used
type reconcilerRegistry map[string]*centralReconciler.CentralReconciler

var reconciledCentralCountCache int32

var cachedOperatorConfigs operator.OperatorConfigs

var backoff = wait.Backoff{
	Duration: 1 * time.Second,
	Factor:   1.5,
	Jitter:   0.1,
	Steps:    15,
	Cap:      10 * time.Minute,
}

// Runtime represents the runtime to reconcile all centrals associated with the given cluster.
type Runtime struct {
	config            *config.Config
	client            *fleetmanager.Client
	clusterID         string
	reconcilers       reconcilerRegistry
	k8sClient         ctrlClient.Client
	dbProvisionClient cloudprovider.DBClient
	statusResponseCh  chan private.DataPlaneCentralStatus
	operatorManager   *operator.ACSOperatorManager
	secretCipher      cipher.Cipher
}

// NewRuntime creates a new runtime
func NewRuntime(config *config.Config, k8sClient ctrlClient.Client) (*Runtime, error) {
	authOption := fleetmanager.Option{
		Sso: fleetmanager.RHSSOOption{
			ClientID:     config.RHSSOClientID,
			ClientSecret: config.RHSSOClientSecret, // pragma: allowlist secret
			Realm:        config.RHSSORealm,
			Endpoint:     config.RHSSOEndpoint,
		},
		Ocm: fleetmanager.OCMOption{
			RefreshToken: config.OCMRefreshToken,
		},
		Static: fleetmanager.StaticOption{
			StaticToken: config.StaticToken,
		},
	}
	auth, err := fleetmanager.NewAuth(config.AuthType, authOption)
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

	operatorManager := operator.NewACSOperatorManager(k8sClient)
	secretCipher, err := cipher.NewCipher(config)
	if err != nil {
		return nil, fmt.Errorf("creating secretCipher: %w", err)
	}

	return &Runtime{
		config:            config,
		k8sClient:         k8sClient,
		client:            client,
		clusterID:         config.ClusterID,
		dbProvisionClient: dbProvisionClient,
		reconcilers:       make(reconcilerRegistry),
		operatorManager:   operatorManager,
		secretCipher:      secretCipher, // pragma: allowlist secret
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

	reconcilerOpts := centralReconciler.CentralReconcilerOptions{
		UseRoutes:         routesAvailable,
		WantsAuthProvider: r.config.CreateAuthProvider,
		EgressProxyImage:  r.config.EgressProxyImage,
		ManagedDBEnabled:  r.config.ManagedDB.Enabled,
		Telemetry:         r.config.Telemetry,
		ClusterName:       r.config.ClusterName,
		Environment:       r.config.Environment,
		AuditLogging:      r.config.AuditLogging,
	}

	ticker := concurrency.NewRetryTicker(func(ctx context.Context) (timeToNextTick time.Duration, err error) {
		list, _, err := r.client.PrivateAPI().GetCentrals(ctx, r.clusterID)
		if err != nil {
			err = errors.Wrapf(err, "retrieving list of managed centrals")
			glog.Error(err)
			return 0, err
		}

		if features.TargetedOperatorUpgrades.Enabled() {
			err := r.upgradeOperator(list)
			if err != nil {
				err = errors.Wrapf(err, "Upgrading operator")
				glog.Error(err)
				return 0, err
			}
		}

		// Start for each Central its own reconciler which can be triggered by sending a central to the receive channel.
		reconciledCentralCountCache = int32(len(list.Items))
		logger.InfoChangedInt32(&reconciledCentralCountCache, "Received central count changed: received %d centrals", reconciledCentralCountCache)
		for _, central := range list.Items {
			if _, ok := r.reconcilers[central.Id]; !ok {
				r.reconcilers[central.Id] = centralReconciler.NewCentralReconciler(r.k8sClient, r.client, central,
					r.dbProvisionClient, postgres.InitializeDatabase, r.secretCipher, reconcilerOpts)
			}

			reconciler := r.reconcilers[central.Id]
			go func(reconciler *centralReconciler.CentralReconciler, central private.ManagedCentral) {
				fleetshardmetrics.MetricsInstance().IncActiveCentralReconcilations()
				defer fleetshardmetrics.MetricsInstance().DecActiveCentralReconcilations()

				// a 15 minutes timeout should cover the duration of a Reconcile call, including the provisioning of an RDS database
				ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
				defer cancel()

				status, err := reconciler.Reconcile(ctx, central)
				fleetshardmetrics.MetricsInstance().IncCentralReconcilations()
				r.handleReconcileResult(central, status, err)
			}(reconciler, central)

			reconcilePaused, err := r.isReconcilePaused(ctx, central)
			if err != nil {
				glog.Warningf("Error getting pause annotation status: %v", err)
			} else {
				fleetshardmetrics.MetricsInstance().SetPauseReconcileStatus(central.Id, reconcilePaused)
			}
		}

		fleetshardmetrics.MetricsInstance().SetTotalCentrals(float64(len(r.reconcilers)))
		if reconcilerOpts.ManagedDBEnabled {
			accountQuotas, err := r.dbProvisionClient.GetAccountQuotas(ctx)
			if err != nil {
				glog.Warningf("Error retrieving account quotas: %v", err)
			} else {
				fleetshardmetrics.MetricsInstance().SetDatabaseAccountQuotas(accountQuotas)
			}
		}

		if features.TargetedOperatorUpgrades.Enabled() {
			operatorWithReplicas, err := r.operatorManager.ListVersionsWithReplicas(ctx)
			if err != nil {
				glog.Warningf("Error retrieving operator versions with replicas: %v", err)
			}
			for image, replicas := range operatorWithReplicas {
				healthy := true
				if replicas == 0 {
					healthy = false
				}
				fleetshardmetrics.MetricsInstance().SetOperatorHealthStatus(image, healthy)
			}
		}

		r.deleteStaleReconcilers(&list)
		return r.config.RuntimePollPeriod, nil
	}, 10*time.Minute, backoff)

	err := ticker.Start()
	if err != nil {
		return fmt.Errorf("starting ticker: %w", err)
	}

	return nil
}

func (r *Runtime) handleReconcileResult(central private.ManagedCentral, status *private.DataPlaneCentralStatus, err error) {
	if err != nil {
		if centralReconciler.IsSkippable(err) {
			glog.V(10).Infof("Skip sending the status for central %s/%s: %v", central.Metadata.Namespace, central.Metadata.Name, err)
		} else {
			fleetshardmetrics.MetricsInstance().IncCentralReconcilationErrors()
			glog.Errorf("Unexpected error occurred %s/%s: %s", central.Metadata.Namespace, central.Metadata.Name, err.Error())
		}
		return
	}
	if status == nil {
		glog.Infof("No status update for Central %s/%s", central.Metadata.Namespace, central.Metadata.Name)
		return
	}
	_, err = r.client.PrivateAPI().UpdateCentralClusterStatus(context.TODO(), r.clusterID, map[string]private.DataPlaneCentralStatus{
		central.Id: *status,
	})
	if err != nil {
		err = errors.Wrapf(err, "updating status for Central %s/%s", central.Metadata.Namespace, central.Metadata.Name)
		glog.Error(err)
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

func (r *Runtime) upgradeOperator(list private.ManagedCentralList) error {
	ctx := context.Background()
	var desiredOperatorImages []string
	for _, operatorDeployment := range list.RhacsOperators.RHACSOperatorConfigs {
		glog.Infof("Installing Operator version: %s", operatorDeployment.GitRef)
		desiredOperatorImages = append(desiredOperatorImages, operatorDeployment.Image)
	}

	// TODO: Replace with list from API request
	operators := operator.OperatorConfigs{
		Configs: []operator.OperatorConfig{{
			Image:  "quay.io/rhacs-eng/stackrox-operator",
			GitRef: "4.1.0",
		}},
		CRD: operator.CRDConfig{
			GitRef: "4.1.0",
		},
	}

	if reflect.DeepEqual(cachedOperatorConfigs, list.RhacsOperators.RHACSOperatorConfigs) {
		return nil
	}
	cachedOperatorConfigs = operators

	// TODO: comment line in to use the API response for production usage after Fleet-Manager implementation is finished
	// err = r.operatorManager.InstallOrUpgrade(ctx, operator.FromAPIResponse(list.RhacsOperators))
	err := r.operatorManager.InstallOrUpgrade(ctx, operators)
	if err != nil {
		return fmt.Errorf("ensuring initial operator installation failed: %w", err)
	}

	err = r.operatorManager.RemoveUnusedOperators(ctx, desiredOperatorImages)
	if err != nil {
		glog.Warningf("Failed removing unused operators: %v", err)
	}
	return nil
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
