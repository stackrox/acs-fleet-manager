// Package reconciler provides update, delete and create logic for managing Central instances.
package reconciler

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"sync/atomic"

	"github.com/golang/glog"
	openshiftRouteV1 "github.com/openshift/api/route/v1"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/k8s"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/util"
	centralConstants "github.com/stackrox/acs-fleet-manager/internal/dinosaur/constants"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/converters"
	"github.com/stackrox/rox/operator/apis/platform/v1alpha1"
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

	revisionAnnotationKey = "rhacs.redhat.com/revision"
)

// CentralReconciler is a reconciler tied to a one Central instance. It installs, updates and deletes Central instances
// in its Reconcile function.
type CentralReconciler struct {
	client            ctrlClient.Client
	central           private.ManagedCentral
	status            *int32
	lastCentralHash   [16]byte
	useRoutes         bool
	wantsAuthProvider bool
	hasAuthProvider   bool
	routeService      *k8s.RouteService
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

	remoteCentralName := remoteCentral.Metadata.Name
	remoteCentralNamespace := remoteCentral.Metadata.Namespace

	if !changed && r.wantsAuthProvider == r.hasAuthProvider && remoteCentral.RequestStatus == centralConstants.DinosaurRequestStatusReady.String() {
		glog.Infof("Central %s/%s not changed, skipping reconciliation", remoteCentralNamespace, remoteCentralName)
		return r.readyStatus(ctx, remoteCentralNamespace)
	}

	centralMonitoringExposeEndpointEnabled := v1alpha1.ExposeEndpointEnabled

	centralResources, err := converters.ConvertPrivateResourceRequirementsToCoreV1(&remoteCentral.Spec.Central.Resources)
	if err != nil {
		return nil, errors.Wrap(err, "converting Central resources")
	}
	scannerAnalyzerResources, err := converters.ConvertPrivateResourceRequirementsToCoreV1(&remoteCentral.Spec.Scanner.Analyzer.Resources)
	if err != nil {
		return nil, errors.Wrap(err, "converting Scanner Analyzer resources")
	}
	scannerAnalyzerScaling := converters.ConvertPrivateScalingToV1(&remoteCentral.Spec.Scanner.Analyzer.Scaling)
	scannerDbResources, err := converters.ConvertPrivateResourceRequirementsToCoreV1(&remoteCentral.Spec.Scanner.Db.Resources)
	if err != nil {
		return nil, errors.Wrap(err, "converting Scanner DB resources")
	}

	central := &v1alpha1.Central{
		ObjectMeta: metav1.ObjectMeta{
			Name:      remoteCentralName,
			Namespace: remoteCentralNamespace,
			Labels:    map[string]string{k8s.ManagedByLabelKey: k8s.ManagedByFleetshardValue},
		},
		Spec: v1alpha1.CentralSpec{
			Central: &v1alpha1.CentralComponentSpec{
				Exposure: &v1alpha1.Exposure{
					Route: &v1alpha1.ExposureRoute{
						Enabled: pointer.BoolPtr(r.useRoutes),
					},
				},
				Monitoring: &v1alpha1.Monitoring{
					ExposeEndpoint: &centralMonitoringExposeEndpointEnabled,
				},
				DeploymentSpec: v1alpha1.DeploymentSpec{
					Resources: &centralResources,
				},
			},
			Scanner: &v1alpha1.ScannerComponentSpec{
				Analyzer: &v1alpha1.ScannerAnalyzerComponent{
					DeploymentSpec: v1alpha1.DeploymentSpec{
						Resources: &scannerAnalyzerResources,
					},
					Scaling: &scannerAnalyzerScaling,
				},
				DB: &v1alpha1.DeploymentSpec{
					Resources: &scannerDbResources,
				},
			},
		},
	}

	// Check whether auth provider is actually created and this reconciler just is not aware of that.
	if r.wantsAuthProvider && !r.hasAuthProvider {
		glog.Info("Wants no has")
		exists, err := existsRHSSOAuthProvider(ctx, remoteCentral, r.client)
		if err != nil {
			return nil, err
		}
		// If sso.redhat.com auth provider exists, there is no need for admin/password login.
		// We also store whether auth provider exists within reconciler instance to avoid polluting network.
		if exists {
			glog.Infof("Auth provider for %s/%s already exists", remoteCentralNamespace, remoteCentralName)
			r.hasAuthProvider = true
		}
	}

	if r.hasAuthProvider {
		central.Spec.Central.AdminPasswordGenerationDisabled = pointer.BoolPtr(true)
	}

	if remoteCentral.Metadata.DeletionTimestamp != "" {
		deleted, err := r.ensureCentralDeleted(ctx, central)
		if err != nil {
			return nil, errors.Wrapf(err, "delete central %s/%s", remoteCentralNamespace, remoteCentralName)
		}
		if deleted {
			return deletedStatus(), nil
		}
		return nil, nil
	}

	if err := r.ensureNamespaceExists(remoteCentralNamespace); err != nil {
		return nil, errors.Wrapf(err, "unable to ensure that namespace %s exists", remoteCentralNamespace)
	}

	centralExists := true
	existingCentral := v1alpha1.Central{}
	err = r.client.Get(ctx, ctrlClient.ObjectKey{Namespace: remoteCentralNamespace, Name: remoteCentralName}, &existingCentral)
	if err != nil {
		if !apiErrors.IsNotFound(err) {
			return nil, errors.Wrapf(err, "unable to check the existence of central %s/%s", central.GetNamespace(), central.GetName())
		}
		centralExists = false
	}

	if !centralExists {
		central.Annotations = map[string]string{revisionAnnotationKey: "1"}

		glog.Infof("Creating central %s/%s", central.GetNamespace(), central.GetName())
		if err := r.client.Create(ctx, central); err != nil {
			return nil, errors.Wrapf(err, "creating new central %s/%s", remoteCentralNamespace, remoteCentralName)
		}
		glog.Infof("Central %s/%s created", central.GetNamespace(), central.GetName())
	} else {
		// TODO(create-ticket): implement update logic
		glog.Infof("Update central %s/%s", central.GetNamespace(), central.GetName())
		existingCentral.Spec = central.Spec

		err = r.incrementCentralRevision(&existingCentral)
		if err != nil {
			return nil, err
		}
		existingCentral.Spec = *central.Spec.DeepCopy()

		if err := r.client.Update(ctx, &existingCentral); err != nil {
			return nil, errors.Wrapf(err, "updating central %s/%s", central.GetNamespace(), central.GetName())
		}
	}

	if r.useRoutes {
		if err := r.ensureRoutesExist(ctx, remoteCentral); err != nil {
			return nil, errors.Wrapf(err, "updating routes")
		}
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
	// 1. Auth provider is already created
	// 2. OR reconciler creator specified auth provider not to be created
	// 3. OR Central request is in status "Ready" - meaning auth provider should've been initialised earlier
	if r.wantsAuthProvider && !r.hasAuthProvider && remoteCentral.RequestStatus != centralConstants.DinosaurRequestStatusReady.String() {
		err = createRHSSOAuthProvider(ctx, remoteCentral, r.client)
		if err != nil {
			return nil, err
		}
		r.hasAuthProvider = true
	}

	// TODO(create-ticket): When should we create failed conditions for the reconciler?
	return r.readyStatus(ctx, remoteCentralNamespace)
}

func (r *CentralReconciler) readyStatus(ctx context.Context, namespace string) (*private.DataPlaneCentralStatus, error) {
	status := readyStatus()
	if r.useRoutes {
		routesStatuses, err := r.getRoutesStatuses(ctx, namespace)
		if err != nil {
			return nil, err
		}
		status.Routes = routesStatuses
	}
	return status, nil
}

func (r *CentralReconciler) getRoutesStatuses(ctx context.Context, namespace string) ([]private.DataPlaneCentralStatusRoutes, error) {
	reencryptIngress, err := r.routeService.FindReencryptIngress(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("obtaining ingress for reencrypt route: %w", err)
	}
	passthroughIngress, err := r.routeService.FindPassthroughIngress(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("obtaining ingress for passthrough route: %w", err)
	}
	return []private.DataPlaneCentralStatusRoutes{
		getRouteStatus(reencryptIngress),
		getRouteStatus(passthroughIngress),
	}, nil
}

func getRouteStatus(ingress *openshiftRouteV1.RouteIngress) private.DataPlaneCentralStatusRoutes {
	return private.DataPlaneCentralStatusRoutes{
		Domain: ingress.Host,
		Router: ingress.RouterCanonicalHostname,
	}
}

func (r CentralReconciler) ensureCentralDeleted(ctx context.Context, central *v1alpha1.Central) (bool, error) {
	globalDeleted := true
	if r.useRoutes {
		reencryptRouteDeleted, err := r.ensureReencryptRouteDeleted(ctx, central.GetNamespace())
		if err != nil {
			return false, err
		}
		passthroughRouteDeleted, err := r.ensurePassthroughRouteDeleted(ctx, central.GetNamespace())
		if err != nil {
			return false, err
		}

		globalDeleted = globalDeleted && reencryptRouteDeleted && passthroughRouteDeleted
	}

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
		return fmt.Errorf("calculating MD5 from JSON: %w", err)
	}

	r.lastCentralHash = hash
	return nil
}

func (r *CentralReconciler) incrementCentralRevision(central *v1alpha1.Central) error {
	revision, err := strconv.Atoi(central.Annotations[revisionAnnotationKey])
	if err != nil {
		return errors.Wrapf(err, "failed increment central revision %s", central.GetName())
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
	if err != nil {
		// Propagate corev1.Namespace to the caller so that the namespace can be easily created
		return namespace, fmt.Errorf("retrieving resource for namespace %q from Kubernetes: %w", name, err)
	}
	return namespace, nil
}

func (r CentralReconciler) ensureNamespaceExists(name string) error {
	namespace, err := r.getNamespace(name)
	if err != nil {
		if apiErrors.IsNotFound(err) {
			err = r.client.Create(context.Background(), namespace)
			if err != nil {
				return fmt.Errorf("creating namespace %q: %w", name, err)
			}
			return nil
		}
		return fmt.Errorf("getting namespace %s: %w", name, err)
	}
	return nil
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

func (r CentralReconciler) ensureRoutesExist(ctx context.Context, remoteCentral private.ManagedCentral) error {
	err := r.ensureReencryptRouteExists(ctx, remoteCentral)
	if err != nil {
		return err
	}
	return r.ensurePassthroughRouteExists(ctx, remoteCentral)
}

// TODO(ROX-9310): Move re-encrypt route reconciliation to the StackRox operator
func (r CentralReconciler) ensureReencryptRouteExists(ctx context.Context, remoteCentral private.ManagedCentral) error {
	namespace := remoteCentral.Metadata.Namespace
	_, err := r.routeService.FindReencryptRoute(ctx, namespace)
	if err != nil && !apiErrors.IsNotFound(err) {
		return fmt.Errorf("retrieving reencrypt route for namespace %q: %w", namespace, err)
	}

	if apiErrors.IsNotFound(err) {
		err = r.routeService.CreateReencryptRoute(ctx, remoteCentral)
		if err != nil {
			return fmt.Errorf("creating reencrypt route for central %s: %w", remoteCentral.Id, err)
		}
	}

	return nil
}

type routeSupplierFunc func() (*openshiftRouteV1.Route, error)

// TODO(ROX-9310): Move re-encrypt route reconciliation to the StackRox operator
// TODO(ROX-11918): Make hostname configurable on the StackRox operator
func (r CentralReconciler) ensureReencryptRouteDeleted(ctx context.Context, namespace string) (bool, error) {
	return r.ensureRouteDeleted(ctx, func() (*openshiftRouteV1.Route, error) {
		return r.routeService.FindReencryptRoute(ctx, namespace) //nolint:wrapcheck
	})
}

// TODO(ROX-11918): Make hostname configurable on the StackRox operator
func (r *CentralReconciler) ensurePassthroughRouteExists(ctx context.Context, remoteCentral private.ManagedCentral) error {
	namespace := remoteCentral.Metadata.Namespace
	_, err := r.routeService.FindPassthroughRoute(ctx, namespace)
	if err != nil && !apiErrors.IsNotFound(err) {
		return fmt.Errorf("retrieving passthrough route for namespace %q: %w", namespace, err)
	}

	if apiErrors.IsNotFound(err) {
		err = r.routeService.CreatePassthroughRoute(ctx, remoteCentral)
		if err != nil {
			return fmt.Errorf("creating passthrough route for central %s: %w", remoteCentral.Id, err)
		}
	}

	return nil
}

// TODO(ROX-11918): Make hostname configurable on the StackRox operator
func (r CentralReconciler) ensurePassthroughRouteDeleted(ctx context.Context, namespace string) (bool, error) {
	return r.ensureRouteDeleted(ctx, func() (*openshiftRouteV1.Route, error) {
		return r.routeService.FindPassthroughRoute(ctx, namespace) //nolint:wrapcheck
	})
}

func (r *CentralReconciler) ensureRouteDeleted(ctx context.Context, routeSupplier routeSupplierFunc) (bool, error) {
	route, err := routeSupplier()
	if err != nil {
		if apiErrors.IsNotFound(err) {
			return true, nil
		}
		return false, errors.Wrapf(err, "get central route %s/%s", route.GetNamespace(), route.GetName())
	}
	if err := r.client.Delete(ctx, route); err != nil {
		return false, errors.Wrapf(err, "delete central route %s/%s", route.GetNamespace(), route.GetName())
	}
	return false, nil
}

// NewCentralReconciler ...
func NewCentralReconciler(k8sClient ctrlClient.Client, central private.ManagedCentral, useRoutes, createAuthProvider bool) *CentralReconciler {
	return &CentralReconciler{
		client:            k8sClient,
		central:           central,
		status:            pointer.Int32(FreeStatus),
		useRoutes:         useRoutes,
		wantsAuthProvider: createAuthProvider,
		routeService:      k8s.NewRouteService(k8sClient),
	}
}
