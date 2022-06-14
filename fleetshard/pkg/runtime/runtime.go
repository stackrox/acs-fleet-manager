package runtime

import (
	"context"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/fleetshard/config"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/centralreconciler"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/fleetmanager"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stackrox/rox/pkg/concurrency"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

// reconcilerRegistry contains a registry of a reconciler for each Central tenant. The key is the identifier of the
// Central instance.
// TODO(SimonBaeumer): set a unique identifier for the map key, currently the instance name is used
type reconcilerRegistry map[string]*centralreconciler.CentralReconciler

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
	reconcilers      reconcilerRegistry //TODO(yury): remove central instance after deletion
	k8sClient        ctrlClient.Client
	statusResponseCh chan private.DataPlaneCentralStatus
}

// NewRuntime creates a new runtime
func NewRuntime(config *config.Config, k8sClient ctrlClient.Client) (*Runtime, error) {
	client, err := fleetmanager.NewClient(config.FleetManagerEndpoint, config.ClusterID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create fleetmanager client")
	}

	return &Runtime{
		config:      config,
		k8sClient:   k8sClient,
		client:      client,
		reconcilers: make(reconcilerRegistry),
	}, nil
}

// Stop stops the runtime
func (r *Runtime) Stop() {
}

// Start starts the fleetshard runtime and schedules
func (r *Runtime) Start() error {
	glog.Infof("fleetshard runtime started")

	ticker := concurrency.NewRetryTicker(func(ctx context.Context) (timeToNextTick time.Duration, err error) {
		list, err := r.client.GetManagedCentralList()
		if err != nil {
			err = errors.Wrapf(err, "retrieving list of managed centrals")
			glog.Error(err)
			return 0, err
		}

		// Start for each Central its own reconciler which can be triggered by sending a central to the receive channel.
		for _, central := range list.Items {
			if _, ok := r.reconcilers[central.Metadata.Name]; !ok {
				r.reconcilers[central.Metadata.Name] = centralreconciler.NewCentralReconciler(r.k8sClient, central)
			}

			reconciler := r.reconcilers[central.Metadata.Name]
			go func(reconciler *centralreconciler.CentralReconciler, central private.ManagedCentral) {
				glog.Infof("Start reconcile central %s", central.Metadata.Name)
				status, err := reconciler.Reconcile(context.Background(), central)
				r.handleReconcileResult(central, status, err)
			}(reconciler, central)
		}

		return r.config.RuntimePollPeriod, nil
	}, 10*time.Minute, backoff)

	return ticker.Start()
}

func (r *Runtime) handleReconcileResult(central private.ManagedCentral, status *private.DataPlaneCentralStatus, err error) {
	if err != nil {
		glog.Errorf("error occurred %s: %s", central.Metadata.Name, err.Error())
		return
	}
	if status == nil {
		glog.Infof("no status update for Central %s", central.Metadata.Name)
		return
	}

	err = r.client.UpdateStatus(map[string]private.DataPlaneCentralStatus{
		central.Id: *status,
	})
	if err != nil {
		err = errors.Wrapf(err, "updating status for Central %s", central.Metadata.Name)
		glog.Error(err)
	}
}
