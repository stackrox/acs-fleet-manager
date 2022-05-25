package main

import (
	"context"
	"flag"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/fleetshard/pkg/fleetmanager"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"golang.org/x/sys/unix"
	v1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"log"
	"os"
	"os/signal"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	clusterID   = "1234567890abcdef1234567890abcdef"
	devEndpoint = "http://127.0.0.1:8000"
)

/**
- 1. setting up fleet-manager
- 2. calling API to get Centrals/Dinosaurs
- 3. Applying Dinosaurs into the Kubernetes API
- 4. Implement polling
- 5. Report status to fleet-manager
*/
func main() {
	// This is needed to make `glog` believe that the flags have already been parsed, otherwise
	// every log messages is prefixed by an error message stating the the flags haven't been
	// parsed.
	_ = flag.CommandLine.Parse([]string{})

	// Always log to stderr by default
	if err := flag.Set("logtostderr", "true"); err != nil {
		glog.Info("Unable to set logtostderr to true")
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, unix.SIGTERM)

	glog.Info("fleetshard application has been started")

	synchronize()

	sig := <-sigs
	glog.Infof("Caught %s signal", sig)
	glog.Info("fleetshard application has been stopped")
}

func synchronize() {
	client, err := fleetmanager.NewClient(devEndpoint, clusterID)
	if err != nil {
		glog.Fatal("failed to create fleetmanager client", err)
	}

	// TODO(create-ticket): Add filter in the REST Client to only receive a specific state
	list, err := client.GetManagedCentralList()
	if err != nil {
		glog.Fatalf("failed to list centrals for cluster %s: %s", clusterID, err)
	}

	statuses := make(map[string]private.DataPlaneDinosaurStatus)
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

func (r ClusterReconciler) Reconcile(ctx context.Context, remoteCentral private.ManagedDinosaur) (*private.DataPlaneDinosaurStatus, error) {
	remoteNamespace := remoteCentral.Metadata.Namespace
	if err := r.ensureNamespace(remoteNamespace); err != nil {
		return nil, errors.Wrapf(err, "unable to ensure that namespace %s exists", remoteNamespace)
	}

	centralExists := false
	central := &v1alpha1.Central{}
	err := r.client.Get(ctx, ctrlClient.ObjectKey{Namespace: remoteNamespace, Name: remoteCentral.Metadata.Name}, central)
	if err != nil {
		if !apiErrors.IsNotFound(err) {
			return nil, errors.Wrapf(err, "unable to check the existence of central %q", central.GetName())
		}
		centralExists = true
	}

	if !centralExists {
		if err := r.client.Create(ctx, central); err != nil {
			return nil, errors.Wrapf(err, "creating new central %q", remoteCentral.Metadata.Name)
		}
	} else {
		// TODO(yury): implement update logic
		glog.Info("Implement update logic for Centrals")
		//if err := r.client.Update(ctx, central); err != nil {
		//	return errors.Wrapf(err, "updating central %q", remoteCentral.GetName())
		//}
	}

	// TODO(create-ticket): When should we create failed conditions for the reconciler?
	return &private.DataPlaneDinosaurStatus{
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
		log.Fatal("fail", err)
	}

	return &ClusterReconciler{
		client: client,
	}
}
