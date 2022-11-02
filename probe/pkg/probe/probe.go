// Package probe ...
package probe

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/constants"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/dinosaurs/types"
	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
	"github.com/stackrox/acs-fleet-manager/probe/config"
	"github.com/stackrox/rox/pkg/concurrency"
	"github.com/stackrox/rox/pkg/httputil"
)

// Probe executes a probe run against fleet manager.
type Probe struct {
	config             *config.Config
	fleetManagerClient fleetmanager.PublicClient
	httpClient         *http.Client
}

// New creates a new probe.
func New(config *config.Config, fleetManagerClient fleetmanager.PublicClient, httpClient *http.Client) (*Probe, error) {
	return &Probe{
		config:             config,
		fleetManagerClient: fleetManagerClient,
		httpClient:         httpClient,
	}, nil
}

func recordElapsedTime(start time.Time) {
	glog.Infof("elapsed time: %v", time.Since(start))
}

func (p *Probe) newCentralName() (string, error) {
	rnd := make([]byte, 8)
	if _, err := rand.Read(rnd); err != nil {
		return "", errors.Wrapf(err, "reading random bytes for unique central name")
	}
	rndString := hex.EncodeToString(rnd)
	return fmt.Sprintf("%s-%s-%s", p.config.ProbeNamePrefix, p.config.ProbeName, rndString), nil
}

// Execute the probe of the fleet manager API.
func (p *Probe) Execute(ctx context.Context) error {
	glog.Info("probe run has been started")
	defer glog.Info("probe run has ended")
	defer recordElapsedTime(time.Now())

	central, err := p.createCentral(ctx)
	if err != nil {
		return err
	}

	if err := p.verifyCentral(ctx, central); err != nil {
		return err
	}

	return p.deleteCentral(ctx, central)
}

// CleanUp remaining probe resources.
func (p *Probe) CleanUp(ctx context.Context, done concurrency.Signal) error {
	defer done.Signal()

	centralList, _, err := p.fleetManagerClient.GetCentrals(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "could not list centrals")
	}

	for i := range centralList.Items {
		central := centralList.Items[i]
		if central.Owner == fmt.Sprintf("service-account-%s", p.config.RHSSOClientID) &&
			strings.HasPrefix(central.Name, fmt.Sprintf("%s-%s", p.config.ProbeNamePrefix, p.config.ProbeName)) {

			if err := p.deleteCentral(ctx, &central); err != nil {
				glog.Errorf("failed to clean up central instance %s: %s", central.Id, err.Error())
			}
		}
	}
	glog.Info("finished clean up attempt of probe resources")
	return nil
}

// Create a Central and verify that it transitioned to 'ready' state.
func (p *Probe) createCentral(ctx context.Context) (*public.CentralRequest, error) {
	centralName, err := p.newCentralName()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create central name")
	}
	request := public.CentralRequestPayload{
		Name:          centralName,
		MultiAz:       true,
		CloudProvider: p.config.DataCloudProvider,
		Region:        p.config.DataPlaneRegion,
	}
	central, _, err := p.fleetManagerClient.CreateCentral(ctx, true, request)
	glog.Infof("creation of central instance requested")
	if err != nil {
		return nil, errors.Wrap(err, "creation of central instance failed")
	}

	centralResp, err := p.ensureCentralState(ctx, &central, constants.CentralRequestStatusReady.String())
	if err != nil {
		return nil, errors.Wrapf(err, "central instance %s did not reach ready state", central.Id)
	}
	return centralResp, nil
}

// Verify that the Central instance has the expected properties and that the
// Central UI is reachable.
func (p *Probe) verifyCentral(ctx context.Context, centralRequest *public.CentralRequest) error {
	if centralRequest.InstanceType != types.STANDARD.String() {
		return errors.Errorf("central has wrong instance type: expected %s, got %s", types.STANDARD.String(), centralRequest.InstanceType)
	}

	if err := p.pingURL(ctx, centralRequest.CentralUIURL); err != nil {
		return errors.Wrapf(err, "could not reach central UI URL of instance %s", centralRequest.Id)
	}
	return nil
}

// Delete the Central instance and verify that it transitioned to 'deprovision' state.
func (p *Probe) deleteCentral(ctx context.Context, centralRequest *public.CentralRequest) error {
	_, err := p.fleetManagerClient.DeleteCentralById(ctx, centralRequest.Id, true)
	glog.Infof("deletion of central instance %s requested", centralRequest.Id)
	if err != nil {
		return errors.Wrapf(err, "deletion of central instance %s failed", centralRequest.Id)
	}

	_, err = p.ensureCentralState(ctx, centralRequest, constants.CentralRequestStatusDeprovision.String())
	if err != nil {
		return errors.Wrapf(err, "central instance %s did not reach deprovision state", centralRequest.Id)
	}

	err = p.ensureCentralDeleted(ctx, centralRequest)
	if err != nil {
		return errors.Wrapf(err, "central instance %s could not be deleted", centralRequest.Id)
	}
	return nil
}

func (p *Probe) ensureCentralState(ctx context.Context, centralRequest *public.CentralRequest, targetState string) (*public.CentralRequest, error) {
	ticker := time.NewTicker(p.config.ProbePollPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, errors.Wrap(ctx.Err(), "ensure central state timed out")
		case <-ticker.C:
			centralResp, _, err := p.fleetManagerClient.GetCentralById(ctx, centralRequest.Id)
			if err != nil {
				glog.Warningf("central instance %s not reachable: %s", centralRequest.Id, err.Error())
				continue
			}

			if centralResp.Status == targetState {
				glog.Infof("central instance %s is in %q state", centralResp.Id, targetState)
				return &centralResp, nil
			}
		}
	}
}

func (p *Probe) ensureCentralDeleted(ctx context.Context, centralRequest *public.CentralRequest) error {
	ticker := time.NewTicker(p.config.ProbePollPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return errors.Wrap(ctx.Err(), "ensure central deleted timed out")
		case <-ticker.C:
			_, response, err := p.fleetManagerClient.GetCentralById(ctx, centralRequest.Id)
			if err != nil {
				if response != nil && response.StatusCode == http.StatusNotFound {
					glog.Infof("central instance %s has been deleted", centralRequest.Id)
					return nil
				}
				glog.Warningf("central instance %s not reachable: %s", centralRequest.Id, err.Error())
			}
		}
	}
}

func (p *Probe) pingURL(ctx context.Context, url string) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
	if err != nil {
		return errors.Wrap(err, "failed to create request for central UI")
	}
	// `httpClient` retries failed requests if they are retryable via `hashicorp/go-retryablehttp`.
	response, err := p.httpClient.Do(request)
	if err != nil {
		return errors.Wrapf(err, "central UI not reachable")
	}
	defer response.Body.Close()
	if !httputil.Is2xxStatusCode(response.StatusCode) {
		return errors.Errorf("central UI ping did not succeed: %s", response.Status)
	}
	return nil
}
