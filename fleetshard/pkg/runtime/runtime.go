package runtime

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/glog"
	centralReconciler "github.com/stackrox/acs-fleet-manager/fleetshard/pkg/central/reconciler"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/fleetshardmetrics"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/k8s"

	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/fleetshard/config"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
	"github.com/stackrox/rox/pkg/concurrency"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

// reconcilerRegistry contains a registry of a reconciler for each Central tenant. The key is the identifier of the
// Central instance.
// TODO(SimonBaeumer): set a unique identifier for the map key, currently the instance name is used
type reconcilerRegistry map[string]*centralReconciler.CentralReconciler

var backoff = wait.Backoff{
	Duration: 1 * time.Second,
	Factor:   1.5,
	Jitter:   0.1,
	Steps:    15,
	Cap:      10 * time.Minute,
}

// Runtime represents the runtime to reconcile all centrals associated with the given cluster.
type Runtime struct {
	config           *config.Config
	client           *fleetmanager.Client
	clusterID        string
	reconcilers      reconcilerRegistry // TODO(create-ticket): possible leak. consider reconcilerRegistry cleanup
	k8sClient        ctrlClient.Client
	statusResponseCh chan private.DataPlaneCentralStatus
}

// NewRuntime creates a new runtime
func NewRuntime(config *config.Config, k8sClient ctrlClient.Client) (*Runtime, error) {
	auth, err := fleetmanager.NewAuth(config.AuthType, fleetmanager.Option{
		Sso: fleetmanager.RHSSOOption{
			ClientID:     config.RHSSOClientID,
			ClientSecret: config.RHSSOClientSecret, //pragma: allowlist secret
			Realm:        config.RHSSORealm,
			Endpoint:     config.RHSSOEndpoint,
		},
		Ocm: fleetmanager.OCMOption{
			RefreshToken: config.OCMRefreshToken,
		},
		Static: fleetmanager.StaticOption{
			StaticToken: config.StaticToken,
		},
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create fleet manager authentication")
	}
	client, err := fleetmanager.NewClient(config.FleetManagerEndpoint, auth, fleetmanager.WithUserAgent(
		fmt.Sprintf("fleetshard-synchronizer/%s", config.ClusterID)),
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create fleet manager client")
	}

	return &Runtime{
		config:      config,
		k8sClient:   k8sClient,
		client:      client,
		clusterID:   config.ClusterID,
		reconcilers: make(reconcilerRegistry),
	}, nil
}

// Stop stops the runtime
func (r *Runtime) Stop() {
}

// Start starts the fleetshard runtime and schedules
func (r *Runtime) Start() error {
	glog.Info("fleetshard runtime started")
	glog.Infof("Auth provider initialisation enabled: %v", r.config.CreateAuthProvider)

	routesAvailable := routesAvailable()

	reconcilerOpts := centralReconciler.CentralReconcilerOptions{
		UseRoutes:         routesAvailable,
		WantsAuthProvider: r.config.CreateAuthProvider,
		EgressProxyImage:  r.config.EgressProxyImage,
	}

	ticker := concurrency.NewRetryTicker(func(ctx context.Context) (timeToNextTick time.Duration, err error) {
		list, _, err := r.client.PrivateAPI().GetCentrals(ctx, r.clusterID)
		if err != nil {
			err = errors.Wrapf(err, "retrieving list of managed centrals")
			glog.Error(err)
			return 0, err
		}

		// Start for each Central its own reconciler which can be triggered by sending a central to the receive channel.
		glog.Infof("Received %d centrals", len(list.Items))
		for _, central := range list.Items {
			if _, ok := r.reconcilers[central.Id]; !ok {
				r.reconcilers[central.Id] = centralReconciler.NewCentralReconciler(r.k8sClient, central, reconcilerOpts)
			}

			reconciler := r.reconcilers[central.Id]
			go func(reconciler *centralReconciler.CentralReconciler, central private.ManagedCentral) {
				fleetshardmetrics.MetricsInstance().IncActiveCentralReconcilations()
				defer fleetshardmetrics.MetricsInstance().DecActiveCentralReconcilations()
				glog.Infof("Start reconcile central %s/%s", central.Metadata.Namespace, central.Metadata.Name)
				status, err := reconciler.Reconcile(context.Background(), central)
				fleetshardmetrics.MetricsInstance().IncCentralReconcilations()
				r.handleReconcileResult(central, status, err)
			}(reconciler, central)
		}
		fleetshardmetrics.MetricsInstance().SetTotalCentrals(float64(len(r.reconcilers)))

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

func routesAvailable() bool {
	available, err := k8s.IsRoutesResourceEnabled()
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
