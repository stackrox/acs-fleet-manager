// Package centralreconciler provides update, delete and create logic for managing Central instances.
package centralreconciler

import (
	"context"
	"fmt"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	v1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
	"sync/atomic"
)

const (
	FreeStatus int32 = iota
	BlockedStatus

	revisionAnnotationKey = "rhacs.redhat.com/revision"
	k8sManagedByLabelKey  = "app.kubernetes.io/managed-by"
)

// CentralReconciler is a reconciler tied to a one Central instance. It installs, updates and deletes Central instances
// in its Reconcile function.
type CentralReconciler struct {
	client  ctrlClient.Client
	central private.ManagedCentral
	status  *int32
}

// Reconcile takes a private.ManagedCentral and tries to install it into the cluster managed by the fleet-shard.
// It tries to create a namespace for the Central and applies necessary updates to the resource.
// TODO(create-ticket): Check correct Central gets reconciled
// TODO(create-ticket): Should an initial ManagedCentral be added on reconciler creation?
// TODO(create-ticket): Add cache to only reconcile if a change to the ManagedCentral was made
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
			Labels:    map[string]string{k8sManagedByLabelKey: "rhacs-fleetshard"},
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
		central.Annotations = map[string]string{revisionAnnotationKey: "1"}

		glog.Infof("Creating central tenant %s", central.GetName())
		if err := r.client.Create(ctx, central); err != nil {
			return nil, errors.Wrapf(err, "creating new central %q", remoteCentral.Metadata.Name)
		}
	} else {
		// TODO(yury): implement update logic
		glog.Infof("Update central tenant %s", central.GetName())
		glog.Info("Implement update logic for Centrals")

		err = r.incrementCentralRevision(central)
		if err != nil {
			return nil, err
		}

		if err := r.client.Update(ctx, central); err != nil {
			return nil, errors.Wrapf(err, "updating central %q", central.GetName())
		}
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

func (r *CentralReconciler) incrementCentralRevision(central *v1alpha1.Central) error {
	revision, err := strconv.Atoi(central.Annotations[revisionAnnotationKey])
	if err != nil {
		return errors.Wrapf(err, "failed incerement central revision %s", central.GetName())
	}
	revision++
	central.Annotations[revisionAnnotationKey] = fmt.Sprintf("%d", revision)
	return nil
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

func NewCentralReconciler(k8sClient ctrlClient.Client, central private.ManagedCentral) *CentralReconciler {
	return &CentralReconciler{
		client:  k8sClient,
		central: central,
		status:  pointer.Int32(FreeStatus),
	}
}
