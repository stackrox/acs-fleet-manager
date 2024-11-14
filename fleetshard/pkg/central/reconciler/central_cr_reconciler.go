package reconciler

import (
	"context"
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/util"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

// centralCrReconciler provides methods to reconcile the Central CR associated with a tenant
type centralCrReconciler struct {
	client ctrlClient.Client
}

// newCentralCrReconciler creates a CentralCrReconciler using given arguments
func newCentralCrReconciler(k8sClient ctrlClient.Client) *centralCrReconciler {
	return &centralCrReconciler{client: k8sClient}
}

// reconcile create or updates the central CR for the tenant in the cluster so that
// it fits to the remote central object received.
func (r *centralCrReconciler) reconcile(ctx context.Context, remoteCentral *private.ManagedCentral, central *v1alpha1.Central) error {
	remoteCentralName := remoteCentral.Metadata.Name
	remoteCentralNamespace := remoteCentral.Metadata.Namespace

	centralExists := true
	existingCentral := v1alpha1.Central{}
	centralKey := ctrlClient.ObjectKey{Namespace: remoteCentralNamespace, Name: remoteCentralName}
	err := r.client.Get(ctx, centralKey, &existingCentral)
	if err != nil {
		if !apiErrors.IsNotFound(err) {
			return errors.Wrapf(err, "unable to check the existence of central %v", centralKey)
		}
		centralExists = false
	}

	if remoteCentral.Metadata.ExpiredAt != nil {
		if central.GetAnnotations() == nil {
			central.Annotations = map[string]string{}
		}
		central.Annotations[centralExpiredAtKey] = remoteCentral.Metadata.ExpiredAt.Format(time.RFC3339)
	}

	if !centralExists {
		if central.GetAnnotations() == nil {
			central.Annotations = map[string]string{}
		}
		if err := util.IncrementCentralRevision(central); err != nil {
			return errors.Wrapf(err, "incrementing Central %v revision", centralKey)
		}

		glog.Infof("Creating Central %v", centralKey)
		if err := r.client.Create(ctx, central); err != nil {
			return errors.Wrapf(err, "creating new Central %v", centralKey)
		}
		glog.Infof("Central %v created", centralKey)
	} else {
		// perform a dry run to see if the update would change anything.
		// This would apply the defaults and the mutating webhooks without actually updating the object.
		// We can then compare the existing object with the object that would be resulting from the update.
		// This will prevent unnecessary operator reconciliation loops.

		desiredCentral := existingCentral.DeepCopy()
		desiredCentral.Spec = *central.Spec.DeepCopy()
		mergeLabelsAndAnnotations(central, desiredCentral)

		requiresUpdate, err := centralNeedsUpdating(ctx, r.client, &existingCentral, desiredCentral)
		if err != nil {
			return errors.Wrapf(err, "checking if Central %v needs to be updated", centralKey)
		}

		if !requiresUpdate {
			glog.Infof("Central %v is already up to date", centralKey)
			return nil
		}

		if err := util.IncrementCentralRevision(desiredCentral); err != nil {
			return errors.Wrapf(err, "incrementing Central %v revision", centralKey)
		}

		if err := r.client.Update(context.Background(), desiredCentral); err != nil {
			return errors.Wrapf(err, "updating Central %v", centralKey)
		}

	}

	return nil
}

// ensureDeleted deletes the central CR from the cluster
func (r *centralCrReconciler) ensureDeleted(ctx context.Context, namespace, crName string) (bool, error) {
	centralKey := ctrlClient.ObjectKey{
		Namespace: namespace,
		Name:      crName,
	}

	err := wait.PollUntilContextCancel(ctx, centralDeletePollInterval, true, func(ctx context.Context) (bool, error) {
		var centralToDelete v1alpha1.Central

		if err := r.client.Get(ctx, centralKey, &centralToDelete); err != nil {
			if apiErrors.IsNotFound(err) {
				return true, nil
			}
			return false, errors.Wrapf(err, "failed to get central CR %v", centralKey)
		}

		// avoid being stuck in a deprovisioning state due to the pause reconcile annotation
		if err := r.disablePauseReconcileIfPresent(ctx, &centralToDelete); err != nil {
			return false, err
		}

		if centralToDelete.GetDeletionTimestamp() == nil {
			glog.Infof("Marking Central CR %v for deletion", centralKey)
			if err := r.client.Delete(ctx, &centralToDelete); err != nil {
				if apiErrors.IsNotFound(err) {
					return true, nil
				}
				return false, errors.Wrapf(err, "failed to delete central CR %v", centralKey)
			}
		}

		glog.Infof("Waiting for Central CR %v to be deleted", centralKey)
		return false, nil
	})

	if err != nil {
		return false, errors.Wrapf(err, "waiting for central CR %v to be deleted", centralKey)
	}
	glog.Infof("Central CR %v is deleted", centralKey)
	return true, nil
}

func (r *centralCrReconciler) disablePauseReconcileIfPresent(ctx context.Context, central *v1alpha1.Central) error {
	if central.Annotations == nil {
		return nil
	}

	if value, exists := central.Annotations[PauseReconcileAnnotation]; !exists || value != "true" {
		return nil
	}

	central.Annotations[PauseReconcileAnnotation] = "false"
	err := r.client.Update(ctx, central)
	if err != nil {
		return fmt.Errorf("removing pause reconcile annotation: %v", err)
	}

	return nil
}
