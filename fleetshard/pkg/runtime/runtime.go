package runtime

import (
	"context"
	"github.com/golang/glog"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/centralreconciler"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/fleetmanager"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stackrox/rox/pkg/concurrency"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

type reconcilerRegistry map[string]*centralreconciler.CentralReconciler

var backoff = wait.Backoff{
	Duration: 5 * time.Second,
	Factor:   3.0,
	Jitter:   0.1,
	Steps:    5,
	Cap:      10 * time.Minute,
}

type Runtime struct {
	client             *fleetmanager.Client
	reconcilers        reconcilerRegistry //TODO(yury): remove central instance after deletion
	k8sClient          ctrlClient.Client
	reconcilerResultCh chan centralreconciler.ReconcilerResult
	statusResponseCh   chan private.DataPlaneCentralStatus
}

func NewRuntime(devEndpoint string, clusterID string, k8sClient ctrlClient.Client) *Runtime {
	client, err := fleetmanager.NewClient(devEndpoint, clusterID)
	if err != nil {
		glog.Fatal("failed to create fleetmanager client", err)
	}

	return &Runtime{
		k8sClient:          k8sClient,
		client:             client,
		reconcilerResultCh: make(chan centralreconciler.ReconcilerResult),
		reconcilers:        make(reconcilerRegistry),
	}
}

func (r *Runtime) Stop() {
}

func (r *Runtime) Start() error {
	glog.Info("fleetshard runtime started")

	go r.watchReconcilerResults()

	ticker := concurrency.NewRetryTicker(func(ctx context.Context) (timeToNextTick time.Duration, err error) {
		list, err := r.client.GetManagedCentralList()
		if err != nil {
			glog.Errorf("failed to list central", err)
			return 0, err
		}

		// Start for each Central its own reconciler which can be triggered by sending a central to the receive channel.
		for _, central := range list.Items {
			if _, ok := r.reconcilers[central.Metadata.Name]; !ok {
				r.reconcilers[central.Metadata.Name] = centralreconciler.NewCentralReconciler(r.k8sClient, r.client, central, r.reconcilerResultCh)
				go r.reconcilers[central.Metadata.Name].Start()
			}

			reconciler := r.reconcilers[central.Metadata.Name]
			reconciler.ReceiveCh() <- &central
		}

		return 1 * time.Second, nil
	}, 10*time.Minute, backoff)

	return ticker.Start()
}

func (r *Runtime) watchReconcilerResults() {
	for result := range r.reconcilerResultCh {
		if result.Err != nil {
			glog.Errorf("error occurred %s: %s", result.Central.Metadata.Name, result)
			continue
		}

		resp, err := r.client.UpdateStatus(map[string]private.DataPlaneCentralStatus{
			result.Central.Id: result.Status,
		})
		if err != nil {
			glog.Errorf("error occurred %s: %s", result.Central.Metadata.Name, result)
		}
		glog.Infof(string(resp))
	}
}
