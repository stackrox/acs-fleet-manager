package centralreconciler

import (
	"context"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/fleetmanager"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	v1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

func Synchronize(devEndpoint string, clusterID string) {
	client, err := fleetmanager.NewClient(devEndpoint, clusterID)
	if err != nil {
		glog.Fatal("failed to create fleetmanager client", err)
	}

	// TODO(create-ticket): Add filter in the REST Client to only receive a specific state
	list, err := client.GetManagedCentralList()
	if err != nil {
		glog.Fatalf("failed to list centrals for cluster %s: %s", clusterID, err)
	}

	statuses := make(map[string]private.DataPlaneCentralStatus)
	for _, remoteCentral := range list.Items {
		glog.Infof("received cluster: %s", remoteCentral.Metadata.Name)

		reconciler := NewClusterReconciler()
		status, err := reconciler.Reconcile(context.Background(), remoteCentral)
		if err != nil {
			glog.Fatalf("failed to reconcile central %s: %s", remoteCentral.Metadata.Name, err)
		}

		statuses[remoteCentral.Id] = *status
	}

	resp, err := client.UpdateStatus(statuses)
	if err != nil {
		glog.Fatal(err)
	}
	glog.Infof(string(resp))
}

// ClusterReconciler reconciles the central cluster
type ClusterReconciler struct {
	client ctrlClient.Client
}

func (r ClusterReconciler) Reconcile(ctx context.Context, remoteCentral private.ManagedCentral) (*private.DataPlaneCentralStatus, error) {
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

func (r ClusterReconciler) ensureNamespace(name string) error {
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

func NewClusterReconciler() *ClusterReconciler {
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = v1alpha1.AddToScheme(scheme)
	config := ctrl.GetConfigOrDie()
	client, err := ctrlClient.New(config, ctrlClient.Options{
		Scheme: scheme,
	})
	if err != nil {
		glog.Fatal("fail", err)
	}

	glog.Infof("Connected to k8s cluster: %s", config.Host)

	return &ClusterReconciler{
		client: client,
	}
}
