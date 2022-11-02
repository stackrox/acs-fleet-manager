// Package runtime ...
package runtime

import (
	"context"
	"net/http"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/pkg/client/fleetmanager"
	"github.com/stackrox/acs-fleet-manager/probe/config"
	"github.com/stackrox/acs-fleet-manager/probe/pkg/probe"
	"github.com/stackrox/rox/pkg/concurrency"
)

// Runtime orchestrates probe runs against fleet manager.
type Runtime struct {
	Config *config.Config
	probe  *probe.Probe
}

// New creates a new runtime.
func New(config *config.Config, fleetManagerClient fleetmanager.PublicClient, httpClient *http.Client) (*Runtime, error) {
	probe, err := probe.New(config, fleetManagerClient, httpClient)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create probe")
	}

	return &Runtime{
		Config: config,
		probe:  probe,
	}, nil
}

// RunLoop a continuous loop of probe runs.
func (r *Runtime) RunLoop(ctx context.Context) error {
	ticker := time.NewTicker(r.Config.ProbeRunWaitPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return errors.Wrap(ctx.Err(), "probe context invalid")
		case <-ticker.C:
			if err := r.RunSingle(ctx); err != nil {
				glog.Warning(err)
			}
		}
	}
}

// RunSingle executes a single probe run.
func (r *Runtime) RunSingle(ctx context.Context) error {
	probeRunCtx, cancel := context.WithTimeout(ctx, r.Config.ProbeRunTimeout)
	defer cancel()
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), r.Config.ProbeCleanUpTimeout)
		defer cancel()
		cleanupDone := concurrency.NewSignal()
		go func() {
			if err := r.probe.CleanUp(cleanupCtx, cleanupDone); err != nil {
				glog.Error(err)
			}
		}()
		select {
		case <-cleanupCtx.Done():
		case <-cleanupDone.Done():
		}
	}()

	if err := r.probe.Execute(probeRunCtx); err != nil {
		return errors.Wrap(err, "probe run failed")
	}
	return nil
}
