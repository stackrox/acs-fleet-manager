package runtime

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/constants"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/probe/config"
	"github.com/stackrox/acs-fleet-manager/probe/pkg/fleetmanager"
	"github.com/stackrox/acs-fleet-manager/probe/pkg/httpclient"
	"github.com/stackrox/acs-fleet-manager/probe/pkg/metrics"
)

// Runtime performs a probe run against fleet manager.
type Runtime struct {
	config             *config.Config
	fleetManagerClient fleetmanager.Client
	request            *public.CentralRequestPayload
	centralState       *public.CentralRequest
}

// New creates a new runtime.
func New(config *config.Config, fleetManagerClient fleetmanager.Client) (*Runtime, error) {
	centralName, err := newCentralName()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create runtime")
	}

	request := &public.CentralRequestPayload{
		Name:          centralName,
		MultiAz:       true,
		CloudProvider: config.DataCloudProvider,
		Region:        config.DataPlaneRegion,
	}

	return &Runtime{
		config:             config,
		fleetManagerClient: fleetManagerClient,
		request:            request,
	}, nil
}

// Run sets up time outs and metrics, and then executes either a single probe run
// or a continuous loop of probe runs.
func (r *Runtime) Run(isErr chan bool, sigs chan os.Signal, loop bool) {
	for {
		ctxTimeout, cancel := context.WithTimeout(context.Background(), r.config.RuntimeRunTimeout)
		defer cancel()
		isDone := make(chan bool, 1)

		go func() {
			metrics.MetricsInstance().IncStartedRuns()
			err := r.probeCentral(ctxTimeout)
			if err != nil {
				metrics.MetricsInstance().IncFailedRuns()
				metrics.MetricsInstance().SetLastFailureTimestamp()
				glog.Error(err)
			} else {
				metrics.MetricsInstance().IncSuccessfulRuns()
				metrics.MetricsInstance().SetLastSuccessTimestamp()
			}
			isDone <- true
		}()

		select {
		case <-ctxTimeout.Done():
			glog.Errorf("Probe run timed out: %v", ctxTimeout.Err())
			metrics.MetricsInstance().IncFailedRuns()
			if loop {
				time.Sleep(r.config.RuntimeRunWaitPeriod)
				continue
			} else {
				isErr <- true
				close(sigs)
				return
			}
		case <-isDone:
			if loop {
				time.Sleep(r.config.RuntimeRunWaitPeriod)
				continue
			} else {
				isErr <- false
				close(sigs)
				return
			}
		}
	}
}

// probeCentral represents a single probe run.
func (r *Runtime) probeCentral(ctx context.Context) error {
	defer totalDurationTimer()()

	err := r.createCentral(ctx)
	if err != nil {
		return err
	}

	err = r.verifyCentral(ctx)
	if err != nil {
		return err
	}

	err = r.deleteCentral(ctx)
	return err
}

// Stop the probe run.
// TODO: Add write ahead log to clean up after ungraceful restarts.
func (r *Runtime) Stop() error {
	if r.centralState == nil {
		return nil
	}
	ctx := context.Background()
	_, err := r.fleetManagerClient.DeleteCentralById(ctx, r.centralState.Id, true)
	if err != nil {
		return errors.Wrapf(err, "Clean up: Deletion of Central instance %s failed", r.centralState.Id)
	}
	glog.Infof("Clean up: Deletion of Central instance %s requested.", r.centralState.Id)
	return nil
}

// totalDurationTimer returns a function that measures the elapsed time between the
// call to the timer and the call to the returned function. The result is tracked
// as a Prometheus metric.
func totalDurationTimer() func() {
	start := time.Now()
	return func() {
		metrics.MetricsInstance().ObserveTotalDuration(time.Since(start))
	}
}

// Create a Central and verify that it transitioned to 'ready' state.
func (r *Runtime) createCentral(ctx context.Context) error {
	central, response, err := r.fleetManagerClient.CreateCentral(ctx, true, *r.request)
	glog.Infof("Creation of Central instance requested.")
	if err != nil {
		return errors.Wrap(err, "Creation of Central instance failed")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusAccepted {
		return errors.Errorf("Creation request to Central instance was not accepted: %s.", response.Status)
	}

	r.centralState = &central
	err = r.ensureCentralState(constants.CentralRequestStatusReady.String())
	return err
}

// Verify that the Central instance has the expected properties and that the
// Central UI is reachable.
func (r *Runtime) verifyCentral(ctx context.Context) error {
	if r.centralState.InstanceType != fleetmanager.StandardInstanceType {
		return errors.Errorf("Central has wrong instance type. Expected %s, got %s.", fleetmanager.StandardInstanceType, r.centralState.InstanceType)
	}

	err := r.pingURL(r.centralState.CentralUIURL)
	return err
}

func (r *Runtime) pingURL(url string) error {
	request, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return errors.Wrapf(err, "Failed to create request for Central UI %s", r.centralState.Id)
	}
	response, err := httpclient.HTTPClient.Do(request)
	if err != nil {
		return errors.Wrapf(err, "Central UI of instance %s not reachable", r.centralState.Id)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return errors.Errorf("Central UI %s did not respond with status OK. Got: %s", r.centralState.Id, response.Status)
	}
	return nil
}

// Delete the Central instance and verify that it transitioned to 'deprovision' state.
func (r *Runtime) deleteCentral(ctx context.Context) error {
	response, err := r.fleetManagerClient.DeleteCentralById(ctx, r.centralState.Id, true)
	glog.Infof("Deletion of Central instance %s requested.", r.centralState.Id)
	if err != nil {
		return errors.Wrapf(err, "Deletion of Central instance %s failed", r.centralState.Id)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusAccepted {
		return errors.Errorf("Deletion request to Central instance %s was not accepted: %s.", r.centralState.Id, response.Status)
	}

	err = r.ensureCentralState(constants.CentralRequestStatusDeprovision.String())
	return err
}

func newCentralName() (string, error) {
	rnd := make([]byte, 8)
	_, err := rand.Read(rnd)
	if err != nil {
		return "", errors.Wrapf(err, "Reading random bytes for unique central name")
	}
	rndString := hex.EncodeToString(rnd)

	return fmt.Sprintf("probe-%s", rndString), nil
}

func (r *Runtime) ensureCentralState(targetState string) error {
	ctxTimeout, cancel := context.WithTimeout(context.Background(), r.config.RuntimePollTimeout)
	defer cancel()

	isDone := make(chan bool, 1)

	go r.pollCentral(ctxTimeout, isDone, targetState)

	select {
	case <-ctxTimeout.Done():
		return errors.Wrapf(ctxTimeout.Err(), "Central instance %s did not reach %s state", r.centralState.Id, targetState)
	case <-isDone:
		return nil
	}
}

func (r *Runtime) pollCentral(ctx context.Context, isDone chan bool, targetState string) error {
	for {
		central, response, err := r.fleetManagerClient.GetCentralById(ctx, r.centralState.Id)
		if err != nil {
			glog.Warningf("Central instance %s not reachable: %s.", r.centralState.Id, err.Error())
			continue
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			glog.Warningf("Central instance %s did not respond with status OK: %s.", r.centralState.Id, response.Status)
			continue
		}

		r.centralState = &central
		if r.centralState.Status == targetState {
			glog.Infof("Central instance %s is in `%s` state.", r.centralState.Id, targetState)
			isDone <- true
			return nil
		}
		time.Sleep(r.config.RuntimePollPeriod)
	}
}
