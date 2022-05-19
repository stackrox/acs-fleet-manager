package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/private"
	"github.com/stackrox/rox/operator/apis/platform/v1alpha1"
	"golang.org/x/sys/unix"
	"io/ioutil"
	v1 "k8s.io/api/core/v1"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"log"
	"net/http"
	"os"
	"os/signal"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ClusterID   = "1234567890abcdef1234567890abcdef"
	DevEndpoint = "http://127.0.0.1:8000"
)

var (
	URLGetCentrals      = fmt.Sprintf("http://127.0.0.1:8000/api/rhacs/v1/agent-clusters/%s/dinosaurs", ClusterID)
	URLPutCentralStatus = fmt.Sprintf("http://127.0.0.1:8000/api/rhacs/v1/agent-clusters/%s/dinosaurs/status", ClusterID)
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
	ocmToken := os.Getenv("OCM_TOKEN")
	if ocmToken == "" {
		glog.Fatal("empty ocm token")
	}

	buf := bytes.Buffer{}
	r, err := http.NewRequest(http.MethodGet, URLGetCentrals, &buf)
	if err != nil {
		glog.Fatal(err)
	}

	r.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ocmToken))
	// TODO: Support pagination
	client := http.Client{}

	glog.Info("Calling the Fleet Manager to get the list of Centrals")

	resp, err := client.Do(r)
	if err != nil {
		glog.Fatal(err)
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		glog.Fatal(err)
	}

	glog.Infof("GOT RESPONSE: %s\n\n", string(respBody))

	list := private.ManagedDinosaurList{}
	err = json.Unmarshal(respBody, &list)
	if err != nil {
		glog.Fatal(err)
	}

	statuses := make(map[string]private.DataPlaneDinosaurStatus)
	for _, v := range list.Items {
		glog.Infof("received cluster: %s", v.Metadata.Name)
		statuses[v.Id] = private.DataPlaneDinosaurStatus{
			Conditions: []private.DataPlaneClusterUpdateStatusRequestConditions{
				{
					Type:   "Ready",
					Status: "True",
				},
			},
		}

		glog.Infof("Calling to update %d statuses %q", len(statuses), URLPutCentralStatus)

		// Create namespace
		reocnciler := NewClusterReconciler()

		// Create resource
		central := &v1alpha1.Central{
			ObjectMeta: metav1.ObjectMeta{
				Name:      v.Metadata.Name,
				Namespace: v.Metadata.Name, //TODO: temporarly use the name
			},
		}

		err = reocnciler.Reconcile(context.Background(), central)
		if err != nil {
			glog.Fatal("NOOO", err)
		}
	}

	// Update request to fleet-manager
	updateBody, err := json.Marshal(statuses)
	if err != nil {
		glog.Fatal(err)
	}

	bufUpdateBody := &bytes.Buffer{}
	bufUpdateBody.Write(updateBody)
	updateReq, err := http.NewRequest(http.MethodPut, URLPutCentralStatus, bufUpdateBody)
	if err != nil {
		glog.Fatal(err)
	}

	updateReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", ocmToken))
	resp, err = client.Do(updateReq)
	if err != nil {
		glog.Fatal(err)
	}
	body, _ := ioutil.ReadAll(resp.Body)

	glog.Infof(string(body))
}

// ClusterReconciler reconciles the central cluster
type ClusterReconciler struct {
	client ctrlClient.Client
}

func (r ClusterReconciler) Reconcile(ctx context.Context, newCentral *v1alpha1.Central) error {
	if err := r.ensureNamespace(newCentral.Namespace); err != nil {
		return errors.Wrapf(err, "unable to ensure that namespace %s exists", newCentral.Namespace)
	}
	centralKey := ctrlClient.ObjectKey{Namespace: newCentral.GetNamespace(), Name: newCentral.GetName()}
	central := &v1alpha1.Central{}
	if err := r.client.Get(ctx, centralKey, central); err != nil {
		if !apiErrors.IsNotFound(err) {
			return errors.Wrapf(err, "unable to check the existence of central %q", newCentral.GetName())
		}
		central = nil
	}

	if central == nil {
		if err := r.client.Create(ctx, central); err != nil {
			return errors.Wrapf(err, "creating new central %q", newCentral.GetName())
		}
	} else {
		newCentral.ResourceVersion = central.ResourceVersion
		// TODO(yury): implement update logic
		if err := r.client.Update(ctx, newCentral); err != nil {
			return errors.Wrapf(err, "updating central %q", newCentral.GetName())
		}
	}

	return nil
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
