// Package centralreconciler provides update, delete and create logic for managing Central instances.
package centralreconciler

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"sync/atomic"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/util"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	FreeStatus int32 = iota
	BlockedStatus

	revisionAnnotationKey = "rhacs.redhat.com/revision"
	k8sManagedByLabelKey  = "app.kubernetes.io/managed-by"
)

// ErrTypeCentralNotChanged is an error returned when reconcilation runs more then once in a row with equal central
var ErrTypeCentralNotChanged = errors.New("central not changed, skipping reconcilation")

// CentralReconciler is a reconciler tied to a one Central instance. It installs, updates and deletes Central instances
// in its Reconcile function.
type CentralReconciler struct {
	client          ctrlClient.Client
	central         private.ManagedCentral
	status          *int32
	lastCentralHash [16]byte
}

// Reconcile takes a private.ManagedCentral and tries to install it into the cluster managed by the fleet-shard.
// It tries to create a namespace for the Central and applies necessary updates to the resource.
// TODO(create-ticket): Check correct Central gets reconciled
// TODO(create-ticket): Should an initial ManagedCentral be added on reconciler creation?
func (r *CentralReconciler) Reconcile(ctx context.Context, remoteCentral private.ManagedCentral) (*private.DataPlaneCentralStatus, error) {
	// Only allow to start reconcile function once
	if !atomic.CompareAndSwapInt32(r.status, FreeStatus, BlockedStatus) {
		return nil, errors.New("Reconciler still busy, skipping reconciliation attempt.")
	}
	defer atomic.StoreInt32(r.status, FreeStatus)

	changed, err := r.centralChanged(remoteCentral)
	if err != nil {
		return nil, errors.Wrapf(err, "checking if central changed")
	}

	if !changed {
		return nil, ErrTypeCentralNotChanged
	}

	remoteCentralName := remoteCentral.Metadata.Name
	remoteNamespace := remoteCentral.Metadata.Namespace

	centralResources, err := convertPrivateResourceRequirementsToCoreV1(&remoteCentral.Spec.Central.Resources)
	if err != nil {
		return nil, errors.Wrap(err, "converting Central resources")
	}
	scannerAnalyzerResources, err := convertPrivateResourceRequirementsToCoreV1(&remoteCentral.Spec.Scanner.Analyzer.Resources)
	if err != nil {
		return nil, errors.Wrap(err, "converting Scanner Analyzer resources")
	}
	scannerAnalyzerScaling, err := convertPrivateScalingToV1(&remoteCentral.Spec.Scanner.Analyzer.Scaling)
	if err != nil {
		return nil, errors.Wrap(err, "converting Scanner Scaling resources")
	}
	scannerDbResources, err := convertPrivateResourceRequirementsToCoreV1(&remoteCentral.Spec.Scanner.Db.Resources)
	if err != nil {
		return nil, errors.Wrap(err, "converting Scanner DB resources")
	}

	central := &v1alpha1.Central{
		ObjectMeta: metav1.ObjectMeta{
			Name:      remoteCentralName,
			Namespace: remoteNamespace,
			Labels:    map[string]string{k8sManagedByLabelKey: "rhacs-fleetshard"},
		},
		Spec: v1alpha1.CentralSpec{
			Central: &v1alpha1.CentralComponentSpec{
				DeploymentSpec: v1alpha1.DeploymentSpec{
					Resources: centralResources,
				},
			},
			Scanner: &v1alpha1.ScannerComponentSpec{
				Analyzer: &v1alpha1.ScannerAnalyzerComponent{
					DeploymentSpec: v1alpha1.DeploymentSpec{
						Resources: scannerAnalyzerResources,
					},
					Scaling: scannerAnalyzerScaling,
				},
				DB: &v1alpha1.DeploymentSpec{
					Resources: scannerDbResources,
				},
			},
		},
	}

	if remoteCentral.Metadata.DeletionTimestamp != "" {
		deleted, err := r.ensureCentralDeleted(context.Background(), central)
		if err != nil {
			return nil, errors.Wrapf(err, "delete central %s", remoteCentralName)
		}
		if deleted {
			return deletedStatus(), nil
		}
		return nil, nil
	}

	if err := r.ensureNamespaceExists(remoteNamespace); err != nil {
		return nil, errors.Wrapf(err, "unable to ensure that namespace %s exists", remoteNamespace)
	}

	centralExists := true
	err = r.client.Get(ctx, ctrlClient.ObjectKey{Namespace: remoteNamespace, Name: remoteCentralName}, central)
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
			return nil, errors.Wrapf(err, "creating new central %s/%s", remoteNamespace, remoteCentralName)
		}
		glog.Infof("Central %s created", central.GetName())
	} else {
		// TODO(create-ticket): implement update logic
		glog.Infof("Update central tenant %s", central.GetName())

		err = r.incrementCentralRevision(central)
		if err != nil {
			return nil, err
		}

		if err := r.client.Update(ctx, central); err != nil {
			return nil, errors.Wrapf(err, "updating central %q", central.GetName())
		}
	}

	if err := r.setLastCentralHash(remoteCentral); err != nil {
		return nil, errors.Wrapf(err, "setting central reconcilation cache")
	}

	// TODO(create-ticket): When should we create failed conditions for the reconciler?
	return readyStatus(), nil
}

func (r CentralReconciler) ensureCentralDeleted(ctx context.Context, central *v1alpha1.Central) (bool, error) {
	if crDeleted, err := r.ensureCentralCRDeleted(ctx, central); err != nil || !crDeleted {
		return false, err
	}
	if namespaceDeleted, err := r.ensureNamespaceDeleted(ctx, central.GetNamespace()); err != nil || !namespaceDeleted {
		return false, err
	}
	glog.Infof("All central resources were deleted: %s/%s", central.GetNamespace(), central.GetName())
	return true, nil
}

// centralChanged compares the given central to the last central reconciled using a hash
func (r *CentralReconciler) centralChanged(central private.ManagedCentral) (bool, error) {
	currentHash, err := util.MD5SumFromJSONStruct(&central)
	if err != nil {
		return true, errors.Wrap(err, "hashing central")
	}

	return !bytes.Equal(r.lastCentralHash[:], currentHash[:]), nil
}

func (r *CentralReconciler) setLastCentralHash(central private.ManagedCentral) error {
	hash, err := util.MD5SumFromJSONStruct(&central)
	if err != nil {
		return err
	}

	r.lastCentralHash = hash
	return nil
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

func (r CentralReconciler) getNamespace(name string) (*v1.Namespace, error) {
	namespace := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	err := r.client.Get(context.Background(), ctrlClient.ObjectKey{Name: name}, namespace)
	return namespace, err
}

func (r CentralReconciler) ensureNamespaceExists(name string) error {
	namespace, err := r.getNamespace(name)
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

func (r CentralReconciler) ensureNamespaceDeleted(ctx context.Context, name string) (bool, error) {
	namespace, err := r.getNamespace(name)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			return true, nil
		}
		return false, errors.Wrapf(err, "delete central namespace %s", name)
	}
	if namespace.Status.Phase == v1.NamespaceTerminating {
		return false, nil // Deletion is already in progress, skipping deletion request
	}
	if err = r.client.Delete(ctx, namespace); err != nil {
		return false, errors.Wrapf(err, "delete central namespace %s", name)
	}
	glog.Infof("Central namespace %s is marked for deletion", name)
	return false, nil
}

func (r CentralReconciler) ensureCentralCRDeleted(ctx context.Context, central *v1alpha1.Central) (bool, error) {
	err := r.client.Get(ctx, ctrlClient.ObjectKey{Namespace: central.GetNamespace(), Name: central.GetName()}, central)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			return true, nil
		}
		return false, errors.Wrapf(err, "delete central CR %s/%s", central.GetNamespace(), central.GetName())
	}
	if err := r.client.Delete(ctx, central); err != nil {
		return false, errors.Wrapf(err, "delete central CR %s/%s", central.GetNamespace(), central.GetName())
	}
	glog.Infof("Central CR %s/%s is marked for deletion", central.GetNamespace(), central.GetName())
	return false, nil
}

func NewCentralReconciler(k8sClient ctrlClient.Client, central private.ManagedCentral) *CentralReconciler {
	return &CentralReconciler{
		client:  k8sClient,
		central: central,
		status:  pointer.Int32(FreeStatus),
	}
}

// We need some helper functions for converting the generated types for `ResourceRequirements` from our own API into the
// proper `ResourceRequirements` type from the Kubernetes core/v1 API, because the latter is used within the `v1alpha1.Central` type,
// while at the same time OpenAPI does not provide a way for referencing official Kubernetes type schemas.

func convertPrivateResourceListToCoreV1(r *private.ResourceList) (corev1.ResourceList, error) {
	var cpuQty resource.Quantity
	var memQty resource.Quantity
	var err error
	if r == nil {
		return corev1.ResourceList{}, nil
	}
	if r.Cpu != "" {
		cpuQty, err = resource.ParseQuantity(r.Cpu)
		if err != nil {
			return corev1.ResourceList{}, errors.Wrapf(err, "parsing CPU quantity %q", r.Cpu)
		}
	}
	if r.Memory != "" {
		memQty, err = resource.ParseQuantity(r.Memory)
		if err != nil {
			return corev1.ResourceList{}, errors.Wrapf(err, "parsing memory quantity %q", r.Memory)
		}
	}
	return corev1.ResourceList{
		corev1.ResourceCPU:    cpuQty,
		corev1.ResourceMemory: memQty,
	}, nil
}

func convertPrivateResourceRequirementsToCoreV1(res *private.ResourceRequirements) (*corev1.ResourceRequirements, error) {
	if res == nil {
		return nil, nil
	}
	requests, err := convertPrivateResourceListToCoreV1(&res.Requests)
	if err != nil {
		return nil, errors.Wrap(err, "parsing resource requests")
	}
	limits, err := convertPrivateResourceListToCoreV1(&res.Limits)
	if err != nil {
		return nil, errors.Wrap(err, "parsing resource limits")
	}
	return &corev1.ResourceRequirements{
		Requests: requests,
		Limits:   limits,
	}, nil
}

func convertPrivateScalingToV1(scaling *private.ManagedCentralAllOfSpecScannerAnalyzerScaling) (*v1alpha1.ScannerAnalyzerScaling, error) {
	if scaling == nil {
		return nil, nil
	}
	autoScaling := scaling.AutoScaling
	return &v1alpha1.ScannerAnalyzerScaling{
		AutoScaling: (*v1alpha1.AutoScalingPolicy)(&autoScaling), // TODO: Validation?
		Replicas:    &scaling.Replicas,
		MinReplicas: &scaling.MinReplicas,
		MaxReplicas: &scaling.MaxReplicas,
	}, nil
}
