// Package reconciler provides update, delete and create logic for managing Central instances.
package reconciler

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"sync/atomic"

	openshiftRouteV1 "github.com/openshift/api/route/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/util"
	centralConstants "github.com/stackrox/acs-fleet-manager/internal/dinosaur/constants"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

// FreeStatus ...
const (
	FreeStatus int32 = iota
	BlockedStatus

	revisionAnnotationKey  = "rhacs.redhat.com/revision"
	k8sManagedByLabelKey   = "app.kubernetes.io/managed-by"
	k8sManagedByLabelValue = "rhacs-fleetshard"

	centralReencryptRouteName = "central-reencrypt"
	centralTLSSecretName      = "central-tls"
)

// ErrTypeCentralNotChanged is an error returned when reconcilation runs more then once in a row with equal central
var ErrTypeCentralNotChanged = errors.New("central not changed, skipping reconcilation")

// CentralReconciler is a reconciler tied to a one Central instance. It installs, updates and deletes Central instances
// in its Reconcile function.
type CentralReconciler struct {
	client             ctrlClient.Client
	central            private.ManagedCentral
	status             *int32
	lastCentralHash    [16]byte
	useRoutes          bool
	createAuthProvider bool
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

	if !changed && !r.createAuthProvider && remoteCentral.RequestStatus == centralConstants.DinosaurRequestStatusReady.String() {
		return nil, ErrTypeCentralNotChanged
	}

	remoteCentralName := remoteCentral.Metadata.Name
	remoteNamespace := remoteCentral.Metadata.Namespace

	central := &v1alpha1.Central{
		ObjectMeta: metav1.ObjectMeta{
			Name:      remoteCentralName,
			Namespace: remoteNamespace,
			Labels:    map[string]string{k8sManagedByLabelKey: k8sManagedByLabelValue},
		},
		Spec: v1alpha1.CentralSpec{
			Central: &v1alpha1.CentralComponentSpec{
				Exposure: &v1alpha1.Exposure{
					Route: &v1alpha1.ExposureRoute{
						Enabled: pointer.BoolPtr(r.useRoutes),
					},
				},
			},
		},
	}

	if remoteCentral.Metadata.DeletionTimestamp != "" {
		deleted, err := r.ensureCentralDeleted(ctx, central)
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

	if err := r.ensureReencryptRouteExists(ctx, remoteCentral); err != nil {
		return nil, errors.Wrapf(err, "updating re-encrypt route")
	}

	if err := r.setLastCentralHash(remoteCentral); err != nil {
		return nil, errors.Wrapf(err, "setting central reconcilation cache")
	}

	// Check whether deployment is ready.
	centralReady, err := isCentralReady(ctx, r.client, remoteCentral)
	if err != nil {
		return nil, err
	}
	if !centralReady {
		return installingStatus(), nil
	}

	// Skip auth provider initialisation if:
	// 1. Auth provider is created by this specific reconciler
	// 2. OR reconciler creator specified auth provider not to be created
	// 3. OR Central request is in status "Ready" - meaning auth provider should've been initialised earlier
	if r.createAuthProvider && remoteCentral.RequestStatus != centralConstants.DinosaurRequestStatusReady.String() {
		err = createRHSSOAuthProvider(ctx, remoteCentral, r.client)
		if err != nil {
			return nil, err
		}
		r.createAuthProvider = false
	}

	// TODO(create-ticket): When should we create failed conditions for the reconciler?
	return readyStatus(), nil
}

func isCentralReady(ctx context.Context, client ctrlClient.Client, central private.ManagedCentral) (bool, error) {
	deployment := &appsv1.Deployment{}
	err := client.Get(ctx,
		ctrlClient.ObjectKey{Name: "central", Namespace: central.Metadata.Namespace},
		deployment)
	if err != nil {
		return false, err
	}
	if deployment.Status.UnavailableReplicas == 0 {
		return true, nil
	}
	return false, nil
}

func (r CentralReconciler) ensureCentralDeleted(ctx context.Context, central *v1alpha1.Central) (bool, error) {
	globalDeleted := true
	routeDeleted, err := r.ensureReencryptRouteDeleted(ctx, central.GetNamespace())
	if err != nil {
		return false, err
	}
	globalDeleted = routeDeleted && globalDeleted

	centralDeleted, err := r.ensureCentralCRDeleted(ctx, central)
	if err != nil {
		return false, err
	}
	globalDeleted = globalDeleted && centralDeleted

	nsDeleted, err := r.ensureNamespaceDeleted(ctx, central.GetNamespace())
	if err != nil {
		return false, err
	}
	globalDeleted = globalDeleted && nsDeleted

	glog.Infof("All central resources were deleted: %s/%s", central.GetNamespace(), central.GetName())
	return globalDeleted, nil
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

func (r CentralReconciler) getNamespace(name string) (*corev1.Namespace, error) {
	namespace := &corev1.Namespace{
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
	if namespace.Status.Phase == corev1.NamespaceTerminating {
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

// TODO(ROX-9310): Move re-encrypt route reconciliation to the StackRox operator
func (r CentralReconciler) ensureReencryptRouteExists(ctx context.Context, remoteCentral private.ManagedCentral) error {
	if !r.useRoutes {
		return nil
	}
	namespace := remoteCentral.Metadata.Namespace
	route := &openshiftRouteV1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      centralReencryptRouteName,
			Namespace: namespace,
			Labels:    map[string]string{k8sManagedByLabelKey: k8sManagedByLabelValue},
		},
	}
	err := r.findRoute(ctx, route)
	if apiErrors.IsNotFound(err) {
		centralTLSSecret := &corev1.Secret{}
		err = r.client.Get(ctx, ctrlClient.ObjectKey{Namespace: namespace, Name: centralTLSSecretName}, centralTLSSecret)
		if err != nil {
			return errors.Wrapf(err, "get central TLS secret %s/%s", namespace, remoteCentral.Metadata.Name)
		}
		centralCA, ok := centralTLSSecret.Data["ca.pem"]
		if !ok {
			return errors.Errorf("could not find centrals ca certificate 'ca.pem' in secret/%s", centralTLSSecretName)
		}
		route.Spec = openshiftRouteV1.RouteSpec{
			Port: &openshiftRouteV1.RoutePort{
				TargetPort: intstr.IntOrString{Type: intstr.String, StrVal: "https"},
			},
			To: openshiftRouteV1.RouteTargetReference{
				Kind: "Service",
				Name: "central",
			},
			TLS: &openshiftRouteV1.TLSConfig{
				Termination:              openshiftRouteV1.TLSTerminationReencrypt,
				Key:                      remoteCentral.Spec.Endpoint.Tls.Key,
				Certificate:              remoteCentral.Spec.Endpoint.Tls.Cert,
				DestinationCACertificate: string(centralCA),
			},
		}
		err = r.client.Create(ctx, route)
	}
	return err
}

func (r CentralReconciler) findRoute(ctx context.Context, route *openshiftRouteV1.Route) error {
	return r.client.Get(ctx, ctrlClient.ObjectKey{Namespace: route.GetNamespace(), Name: route.GetName()}, route)
}

// TODO(ROX-9310): Move re-encrypt route reconciliation to the StackRox operator
func (r CentralReconciler) ensureReencryptRouteDeleted(ctx context.Context, namespace string) (bool, error) {
	if !r.useRoutes {
		return true, nil
	}
	route := &openshiftRouteV1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:      centralReencryptRouteName,
			Namespace: namespace,
			Labels:    map[string]string{k8sManagedByLabelKey: k8sManagedByLabelValue},
		},
	}
	if err := r.findRoute(ctx, route); err != nil {
		if apiErrors.IsNotFound(err) {
			return true, nil
		}
		return false, errors.Wrapf(err, "get central re-encrypt route %s/%s", namespace, route.GetName())
	}
	if err := r.client.Delete(ctx, route); err != nil {
		return false, errors.Wrapf(err, "delete central re-encrypt route %s/%s", namespace, route.GetName())
	}
	return false, nil
}

// NewCentralReconciler ...
func NewCentralReconciler(k8sClient ctrlClient.Client, central private.ManagedCentral, useRoutes, createAuthProvider bool) *CentralReconciler {
	return &CentralReconciler{
		client:             k8sClient,
		central:            central,
		status:             pointer.Int32(FreeStatus),
		useRoutes:          useRoutes,
		createAuthProvider: createAuthProvider,
	}
}
