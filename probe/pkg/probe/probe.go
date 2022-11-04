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
	"github.com/stackrox/rox/pkg/httputil"
)

// Probe is a wrapper interface for the core probe logic.
//
//go:generate moq -out probe_moq.go . Probe
type Probe interface {
	Execute(ctx context.Context) error
	CleanUp(ctx context.Context) error
}

var _ Probe = (*ProbeImpl)(nil)

// ProbeImpl executes a probe run against fleet manager.
type ProbeImpl struct {
	config             *config.Config
	fleetManagerClient fleetmanager.PublicClient
	httpClient         *http.Client
}

// New creates a new probe.
func New(config *config.Config, fleetManagerClient fleetmanager.PublicClient, httpClient *http.Client) *ProbeImpl {
	return &ProbeImpl{
		config:             config,
		fleetManagerClient: fleetManagerClient,
		httpClient:         httpClient,
	}
}

func recordElapsedTime(start time.Time) {
	glog.Infof("elapsed time: %v", time.Since(start))
}

func (p *ProbeImpl) newCentralName() (string, error) {
	rnd := make([]byte, 8)
	if _, err := rand.Read(rnd); err != nil {
		return "", errors.Wrapf(err, "reading random bytes for unique central name")
	}
	rndString := hex.EncodeToString(rnd)
	return fmt.Sprintf("%s-%s-%s", p.config.ProbeNamePrefix, p.config.ProbeName, rndString), nil
}

// Execute the probe of the fleet manager API.
func (p *ProbeImpl) Execute(ctx context.Context) error {
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
func (p *ProbeImpl) CleanUp(ctx context.Context) error {
	if err := retryUntilSucceeded(ctx, p.cleanupFunc, p.config.ProbePollPeriod); err != nil {
		return errors.Wrap(err, "cleanup centrals failed")
	}
	return nil
}

func (p *ProbeImpl) cleanupFunc(ctx context.Context) error {
	centralList, _, err := p.fleetManagerClient.GetCentrals(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "could not list centrals")
	}

	serviceAccountName := fmt.Sprintf("service-account-%s", p.config.RHSSOClientID)
	namePrefix := fmt.Sprintf("%s-%s", p.config.ProbeNamePrefix, p.config.ProbeName)
	success := true
	for _, central := range centralList.Items {
		central := central
		if central.Owner != serviceAccountName || !strings.HasPrefix(central.Name, namePrefix) {
			continue
		}
		if err := p.deleteCentral(ctx, &central); err != nil {
			glog.Warningf("failed to clean up central instance %s: %s", central.Id, err)
			success = false
		}
	}

	if success {
		glog.Info("finished clean up attempt of probe resources")
		return nil
	}
	return errors.New("central clean up not successful")
}

// Create a Central and verify that it transitioned to 'ready' state.
func (p *ProbeImpl) createCentral(ctx context.Context) (*public.CentralRequest, error) {
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
func (p *ProbeImpl) verifyCentral(ctx context.Context, centralRequest *public.CentralRequest) error {
	if centralRequest.InstanceType != types.STANDARD.String() {
		return errors.Errorf("central has wrong instance type: expected %s, got %s", types.STANDARD, centralRequest.InstanceType)
	}

	if err := p.pingURL(ctx, centralRequest.CentralUIURL); err != nil {
		return errors.Wrapf(err, "could not reach central UI URL of instance %s", centralRequest.Id)
	}
	return nil
}

// Delete the Central instance and verify that it transitioned to 'deprovision' state.
func (p *ProbeImpl) deleteCentral(ctx context.Context, centralRequest *public.CentralRequest) error {
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

func (p *ProbeImpl) ensureCentralState(ctx context.Context, centralRequest *public.CentralRequest, targetState string) (*public.CentralRequest, error) {
	ensureFunc := func(context.Context) (*public.CentralRequest, error) {
		centralResp, _, err := p.fleetManagerClient.GetCentralById(ctx, centralRequest.Id)
		if err != nil {
			return nil, errors.Wrapf(err, "central instance %s not reachable", centralRequest.Id)
		}

		if centralResp.Status == targetState {
			glog.Infof("central instance %s is in %q state", centralResp.Id, targetState)
			return &centralResp, nil
		}
		return nil, errors.Errorf("central instance %s not in target state %q", centralRequest.Id, targetState)
	}

	centralResp, err := retryUntilSucceededWithResponse(ctx, ensureFunc, p.config.ProbePollPeriod)
	if err != nil {
		return nil, errors.Wrap(err, "ensure central state failed")
	}
	return centralResp, nil
}

func (p *ProbeImpl) ensureCentralDeleted(ctx context.Context, centralRequest *public.CentralRequest) error {
	ensureFunc := func(context.Context) error {
		_, response, err := p.fleetManagerClient.GetCentralById(ctx, centralRequest.Id)
		if err != nil {
			if response != nil && response.StatusCode == http.StatusNotFound {
				glog.Infof("central instance %s has been deleted", centralRequest.Id)
				return nil
			}
			return errors.Wrapf(err, "central instance %s not reachable", centralRequest.Id)
		}
		return errors.Errorf("central instance %s not deleted", centralRequest.Id)
	}

	if err := retryUntilSucceeded(ctx, ensureFunc, p.config.ProbePollPeriod); err != nil {
		return errors.Wrap(err, "ensure central deleted failed")
	}
	return nil
}

func (p *ProbeImpl) pingURL(ctx context.Context, url string) error {
	pingFunc := func(context.Context) error {
		request, err := http.NewRequestWithContext(ctx, http.MethodHead, url, nil)
		if err != nil {
			return errors.Wrap(err, "failed to create request")
		}
		response, err := p.httpClient.Do(request)
		if err != nil {
			return errors.Wrap(err, "URL not reachable")
		}
		defer response.Body.Close()
		if !httputil.Is2xxStatusCode(response.StatusCode) {
			return errors.Errorf("URL ping did not succeed: %s", response.Status)
		}
		return nil
	}

	if err := retryUntilSucceeded(ctx, pingFunc, p.config.ProbePollPeriod); err != nil {
		return errors.Wrap(err, "URL ping failed")
	}
	return nil
}

func retryUntilSucceeded(ctx context.Context, fn func(context.Context) error, interval time.Duration) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			err := fn(ctx)
			if err != nil {
				glog.Warning(err)
				continue
			}
			return nil
		}
	}
}

func retryUntilSucceededWithResponse(ctx context.Context, fn func(context.Context) (*public.CentralRequest, error), interval time.Duration) (*public.CentralRequest, error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			centralResp, err := fn(ctx)
			if err != nil {
				glog.Warning(err)
				continue
			}
			return centralResp, nil
		}
	}
}
