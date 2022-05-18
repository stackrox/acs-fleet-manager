package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/golang/glog"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
	"golang.org/x/sys/unix"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
)

const URLGetCentrals = "http://127.0.0.1:8000/api/rhacs/v1/centrals"
const URLPutCentralStatus = "http://127.0.0.1:8000/api/rhacs/v1/centrals/%s/status"

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

	callFleetManager()

	sig := <-sigs
	glog.Infof("Caught %s signal", sig)
	glog.Info("fleetshard application has been stopped")
}

func callFleetManager() {
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

	list := public.CentralRequestList{}
	err = json.Unmarshal(respBody, &list)
	if err != nil {
		glog.Fatal(err)
	}

	for _, v := range list.Items {
		glog.Infof("received cluster: %s", v.Name)
		glog.Infof("Calling to update status %q", fmt.Sprintf(URLPutCentralStatus, v.Id))
	}
}
