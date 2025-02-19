// Package probe ...
package probe

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/constants"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/api/public"
	"github.com/stackrox/acs-fleet-manager/internal/dinosaur/pkg/dinosaurs/types"
	"github.com/stackrox/acs-fleet-manager/probe/config"
	"github.com/stackrox/acs-fleet-manager/probe/pkg/central"
	"github.com/stackrox/acs-fleet-manager/probe/pkg/metrics"
)

// Probe executes a probe run against fleet manager.
type Probe struct {
	config         config.Config
	spec           central.Spec
	centralService central.Service
}

// New creates a new probe.
func New(config config.Config, centralService central.Service, spec central.Spec) *Probe {
	return &Probe{
		config:         config,
		centralService: centralService,
		spec:           spec,
	}
}

func (p *Probe) recordElapsedTime(start time.Time) {
	elapsedTime := time.Since(start)
	glog.Infof("elapsed time: %v", elapsedTime)
	metrics.MetricsInstance().ObserveTotalDuration(elapsedTime, p.spec.Region)
}

func (p *Probe) newCentralName() (string, error) {
	rnd := make([]byte, 2)
	if _, err := rand.Read(rnd); err != nil {
		return "", errors.Wrapf(err, "reading random bytes for unique central name")
	}
	rndString := hex.EncodeToString(rnd)
	return fmt.Sprintf("%s-%s", p.config.ProbeName, rndString), nil
}

// Execute the probe of the fleet manager API.
func (p *Probe) Execute(ctx context.Context) error {
	glog.Infof("probe run has been started: fleetManagerEndpoint=%q, provider=%q, region=%q",
		p.config.FleetManagerEndpoint,
		p.spec.CloudProvider,
		p.spec.Region,
	)
	defer glog.Info("probe run has ended")
	defer p.recordElapsedTime(time.Now())
	if err := p.cleanup(ctx); err != nil {
		return err
	}

	central, err := p.createCentral(ctx)
	if err != nil {
		return err
	}
	glog.Info("central creation succeeded; proceeding with verification")

	if err := p.verifyCentral(ctx, central); err != nil {
		return err
	}
	glog.Info("central verification succeeded; proceeding with deletion")

	return p.deleteCentral(ctx, central)
}

func (p *Probe) cleanup(ctx context.Context) error {
	if err := retryUntilSucceeded(ctx, p.cleanupFunc, p.config.ProbePollPeriod); err != nil {
		return errors.Wrap(err, "cleanup centrals failed")
	}
	return nil
}

func (p *Probe) cleanupFunc(ctx context.Context) error {
	centralList, err := p.centralService.List(ctx, p.spec)
	if err != nil {
		return errors.Wrap(err, "cleanup failed")
	}

	centralsLeft := false
	for _, central := range centralList {
		central := central
		// Remove all instances that have been created by the probe user.
		// To avoid intefering with other probe instances, we only remove instances
		// with the prefix of the current instance or orphaned instances.
		// An instance is considered orphaned after 24 hours from creation.
		hasProbeOwner := central.Owner == p.config.ProbeUsername
		hasProbePrefix := strings.HasPrefix(central.Name, p.config.ProbeName)
		isOrphan := time.Now().Sub(central.CreatedAt) > 24*time.Hour
		if !hasProbeOwner || (!hasProbePrefix && !isOrphan) {
			continue
		}
		centralsLeft = true
		if alreadyDeleting(central) {
			continue
		}
		if err := p.centralService.Delete(ctx, central.Id); err != nil {
			glog.Warningf("failed to delete central instance %s: %s", central.Id, err)
		}
	}

	if !centralsLeft {
		glog.Info("finished clean up attempt of probe resources")
		return nil
	}
	return errors.New("central clean up not successful")
}

func alreadyDeleting(central public.CentralRequest) bool {
	status := constants.CentralStatus(central.Status)
	return status == constants.CentralRequestStatusDeprovision || status == constants.CentralRequestStatusDeleting
}

// Create a Central and verify that it transitioned to 'ready' state.
func (p *Probe) createCentral(ctx context.Context) (*public.CentralRequest, error) {
	centralName, err := p.newCentralName()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create central name")
	}
	central, err := p.centralService.Create(ctx, centralName, p.spec)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create central instance")
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
		return errors.Errorf("central has wrong instance type: expected %s, got %s", types.STANDARD, centralRequest.InstanceType)
	}

	if err := p.pingURL(ctx, centralRequest.CentralUIURL); err != nil {
		return errors.Wrapf(err, "could not reach central UI URL of instance %s", centralRequest.Id)
	}
	return nil
}

// Delete the Central instance and make sure it is missing from the Fleet Manager API.
func (p *Probe) deleteCentral(ctx context.Context, centralRequest *public.CentralRequest) error {
	if err := p.centralService.Delete(ctx, centralRequest.Id); err != nil {
		return err
	}
	if err := p.ensureCentralDeleted(ctx, centralRequest); err != nil {
		return errors.Wrapf(err, "central instance %s with status %s could not be deleted", centralRequest.Id, centralRequest.Status)
	}
	glog.Info("central deletion succeeded")
	return nil
}

func (p *Probe) ensureCentralState(ctx context.Context, centralRequest *public.CentralRequest, targetState string) (*public.CentralRequest, error) {
	funcWrapper := func(funcCtx context.Context) (*public.CentralRequest, error) {
		return p.ensureStateFunc(funcCtx, centralRequest, targetState)
	}
	centralResp, err := retryUntilSucceededWithResponse(ctx, funcWrapper, p.config.ProbePollPeriod)
	if err != nil {
		return nil, errors.Wrap(err, "ensure central state failed")
	}
	return centralResp, nil
}

func (p *Probe) ensureStateFunc(ctx context.Context, centralRequest *public.CentralRequest, targetState string) (*public.CentralRequest, error) {
	centralResp, err := p.centralService.Get(ctx, centralRequest.Id)
	if err != nil {
		return nil, errors.Wrapf(err, "ensure state %s for central %s", targetState, centralRequest.Id)
	}

	if centralResp.Status == targetState {
		glog.Infof("central instance %s is in %q state", centralResp.Id, targetState)
		return &centralResp, nil
	}
	err = errors.Errorf("central instance %s not in target state %q", centralRequest.Id, targetState)
	return nil, err
}

func (p *Probe) ensureCentralDeleted(ctx context.Context, centralRequest *public.CentralRequest) error {
	funcWrapper := func(funcCtx context.Context) error {
		return p.ensureDeletedFunc(funcCtx, centralRequest)
	}

	if err := retryUntilSucceeded(ctx, funcWrapper, p.config.ProbePollPeriod); err != nil {
		return errors.Wrap(err, "ensure central deleted failed")
	}
	return nil
}

func (p *Probe) ensureDeletedFunc(ctx context.Context, centralRequest *public.CentralRequest) error {
	_, err := p.centralService.Get(ctx, centralRequest.Id)
	if err != nil {
		if errors.Is(err, central.ErrNotFound) {
			glog.Infof("central instance %s has been deleted", centralRequest.Id)
			return nil
		}
		return errors.Wrapf(err, "central instance %s not deleted", centralRequest.Id)

	}
	err = errors.Errorf("central instance %s not deleted", centralRequest.Id)
	return err
}

func (p *Probe) pingURL(ctx context.Context, url string) error {
	funcWrapper := func(funcCtx context.Context) error {
		return p.centralService.Ping(funcCtx, url)
	}
	if err := retryUntilSucceeded(ctx, funcWrapper, p.config.ProbePollPeriod); err != nil {
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
			return errors.Wrap(ctx.Err(), "retry failed")
		case <-ticker.C:
			if err := fn(ctx); err == nil {
				return nil
			}
		}
	}
}

func retryUntilSucceededWithResponse(ctx context.Context, fn func(context.Context) (*public.CentralRequest, error), interval time.Duration) (*public.CentralRequest, error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, errors.Wrap(ctx.Err(), "retry failed")
		case <-ticker.C:
			if centralResp, err := fn(ctx); err == nil {
				return centralResp, nil
			}
		}
	}
}
