package centralreconciler

import (
	"context"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	v1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sync/atomic"
)

const (
	FreeStatus int32 = iota
	BlockedStatus
)

// CentralReconciler reconciles the central cluster
type CentralReconciler struct {
	client     ctrlClient.Client
	central    private.ManagedCentral
	inputCh    chan *private.ManagedCentral
	resultCh   chan ReconcilerResult
	stopCh     chan struct{}
	status     *int32
	responseCh chan private.DataPlaneCentralStatus
}

// TODO: Correct error creation?
type ReconcilerResult struct {
	Central private.ManagedCentral
	Err     error
	Status  private.DataPlaneCentralStatus
}

// TODO(create-ticket): Setup local watch on Central's kube API resources to set new updates statues?
// TODO(create-ticket): Graceful shutdown?
func (r CentralReconciler) Start() {
	// TODO: prevent multiple starts
	for {
		select {
		case central := <-r.inputCh:
			go func() {
				//TODO: cancellation?
				status, err := r.Reconcile(context.Background(), *central)
				r.resultCh <- ReconcilerResult{Err: err, Status: *status, Central: *central}
			}()
		case <-r.stopCh:
			return
		}
	}
}

func (r CentralReconciler) InputChannel() chan *private.ManagedCentral {
	return r.inputCh
}

func (r CentralReconciler) Reconcile(ctx context.Context, remoteCentral private.ManagedCentral) (*private.DataPlaneCentralStatus, error) {
	// Only allow to start reconcile function once
	if !atomic.CompareAndSwapInt32(r.status, FreeStatus, BlockedStatus) {
		return nil, errors.New("Reconciler still busy, skipping reconciliation attempt.")
	}
	defer atomic.StoreInt32(r.status, FreeStatus)

	remoteNamespace := remoteCentral.Metadata.Name
	if err := r.ensureNamespace(remoteCentral.Metadata.Name); err != nil {
		return nil, errors.Wrapf(err, "unable to ensure that namespace %s exists", remoteNamespace)
	}

	centralExists := true
	central := &v1alpha1.Central{
		ObjectMeta: metav1.ObjectMeta{
			Name:      remoteCentral.Metadata.Name,
			Namespace: remoteNamespace,
		},
	}

	err := r.client.Get(ctx, ctrlClient.ObjectKey{Namespace: remoteNamespace, Name: remoteCentral.Metadata.Name}, central)
	if err != nil {
		if !apiErrors.IsNotFound(err) {
			return nil, errors.Wrapf(err, "unable to check the existence of central %q", central.GetName())
		}
		centralExists = false
	}

	if !centralExists {
		glog.Infof("Creating central tenant %s", central.GetName())
		if err := r.client.Create(ctx, central); err != nil {
			return nil, errors.Wrapf(err, "creating new central %q", remoteCentral.Metadata.Name)
		}
	} else {
		// TODO(yury): implement update logic
		glog.Infof("Update central tenant %s", central.GetName())
		glog.Info("Implement update logic for Centrals")
		//if err := r.client.Update(ctx, central); err != nil {
		//	return errors.Wrapf(err, "updating central %q", remoteCentral.GetName())
		//}
	}

	// TODO(create-ticket): When should we create failed conditions for the reconciler?
	return &private.DataPlaneCentralStatus{
		Conditions: []private.DataPlaneClusterUpdateStatusRequestConditions{
			{
				Type:   "Ready",
				Status: "True",
			},
		},
	}, nil
}

func (r CentralReconciler) ensureNamespace(name string) error {
	namespace := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	err := r.client.Get(context.Background(), ctrlClient.ObjectKey{Name: name}, namespace)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			err = r.client.Create(context.Background(), namespace)
			if err != nil {
				return nil
			}
		}
	}
	return err
}

func NewCentralReconciler(k8sClient ctrlClient.Client, central private.ManagedCentral, resultCh chan ReconcilerResult) *CentralReconciler {
	return &CentralReconciler{
		client:   k8sClient,
		central:  central,
		resultCh: resultCh,
		inputCh:  make(chan *private.ManagedCentral),
		status:   pointer.Int32(FreeStatus),
	}
}
