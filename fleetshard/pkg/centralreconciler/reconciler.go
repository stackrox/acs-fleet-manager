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
	client  ctrlClient.Client
	central private.ManagedCentral
	status  *int32
}

func (r CentralReconciler) Reconcile(ctx context.Context, remoteCentral private.ManagedCentral) (*private.DataPlaneCentralStatus, error) {
	// Only allow to start reconcile function once
	if !atomic.CompareAndSwapInt32(r.status, FreeStatus, BlockedStatus) {
		return nil, errors.New("Reconciler still busy, skipping reconciliation attempt.")
	}
	defer atomic.StoreInt32(r.status, FreeStatus)

	remoteCentralName := remoteCentral.Metadata.Name
	remoteNamespace := remoteCentralName

	central := &v1alpha1.Central{
		ObjectMeta: metav1.ObjectMeta{
			Name:      remoteCentralName,
			Namespace: remoteNamespace,
		},
	}

	if remoteCentral.Metadata.DeletionTimestamp != "" {
		glog.Infof("Deleting central %s", remoteCentralName)
		if err := r.deleteCentral(central); err != nil {
			return nil, errors.Wrapf(err, "delete central %s", remoteCentralName)
		}
		return deletedStatus(), nil
	}

	if err := r.ensureNamespace(remoteCentralName); err != nil {
		return nil, errors.Wrapf(err, "unable to ensure that namespace %s exists", remoteNamespace)
	}

	centralExists := true
	err := r.client.Get(ctx, ctrlClient.ObjectKey{Namespace: remoteNamespace, Name: remoteCentralName}, central)
	if err != nil {
		if !apiErrors.IsNotFound(err) {
			return nil, errors.Wrapf(err, "unable to check the existence of central %q", central.GetName())
		}
		centralExists = false
	}

	if !centralExists {
		glog.Infof("Creating central tenant %s", central.GetName())
		if err := r.client.Create(ctx, central); err != nil {
			return nil, errors.Wrapf(err, "creating new central %q", remoteCentralName)
		}
	} else {
		// TODO(create-ticket): implement update logic
		glog.Infof("Update central tenant %s", central.GetName())
		glog.Info("Implement update logic for Centrals")
		//if err := r.client.Update(ctx, central); err != nil {
		//	return errors.Wrapf(err, "updating central %q", remoteCentral.GetName())
		//}
	}

	// TODO(create-ticket): When should we create failed conditions for the reconciler?
	return readyStatus(), nil
}

func (r CentralReconciler) deleteCentral(central *v1alpha1.Central) error {
	pvcs, err := r.getOwnedPVCs(central)
	// get used PVCs before deleting the CR because operator erases the PVC ownerReference after CR deletion
	if err != nil {
		return errors.Wrapf(err, "get central PVCS %s/%s", central.GetName(), central.GetNamespace())
	}
	if err := r.deleteCentralCR(central); err != nil {
		return errors.Wrapf(err, "delete central CR %s/%s", central.GetName(), central.GetNamespace())
	}
	for _, pvc := range pvcs {
		if err := r.deleteCentralPVC(pvc); err != nil {
			return errors.Wrapf(err, "delete PVC %s/%s", pvc.GetName(), pvc.GetNamespace())
		}
	}
	if err := r.deleteNamespace(central.GetNamespace()); err != nil {
		return errors.Wrapf(err, "delete central namespace %s", central.GetNamespace())
	}
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

func (r CentralReconciler) deleteCentralCR(central *v1alpha1.Central) error {
	return r.client.Delete(context.Background(), central)

}

func (r CentralReconciler) deleteNamespace(name string) error {
	namespace := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	return r.client.Delete(context.Background(), namespace)
}

func (r CentralReconciler) deleteCentralPVC(pvc *v1.PersistentVolumeClaim) error {
	return r.client.Delete(context.Background(), pvc)
}

func (r CentralReconciler) getOwnedPVCs(central *v1alpha1.Central) ([]*v1.PersistentVolumeClaim, error) {
	pvcList := &v1.PersistentVolumeClaimList{}
	if err := r.client.List(context.Background(), pvcList, ctrlClient.InNamespace(central.GetNamespace())); err != nil {
		return nil, errors.Wrapf(err, "receiving list PVC list for %s %s", central.GroupVersionKind(), central.GetName())
	}

	var ownedPVCs []*v1.PersistentVolumeClaim
	for i := range pvcList.Items {
		item := pvcList.Items[i]
		if metav1.IsControlledBy(&item, central) {
			tmp := item
			ownedPVCs = append(ownedPVCs, &tmp)
		}
	}

	return ownedPVCs, nil
}

func NewCentralReconciler(k8sClient ctrlClient.Client, central private.ManagedCentral) *CentralReconciler {
	return &CentralReconciler{
		client:  k8sClient,
		central: central,
		status:  pointer.Int32(FreeStatus),
	}
}
