// Package runtime ...
package runtime

import (
	"context"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/stackrox/acs-fleet-manager/probe/config"
	"github.com/stackrox/acs-fleet-manager/probe/pkg/metrics"
	"github.com/stackrox/acs-fleet-manager/probe/pkg/probe"
)

var (
	errCleanupFailed = errors.New("cleanup failed")
)

// Runtime orchestrates probe runs against fleet manager.
type Runtime struct {
	Config *config.Config
	probe  probe.Probe
}

// New creates a new runtime.
func New(config *config.Config, probe probe.Probe) (*Runtime, error) {
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
				glog.Error(err)
			}
		}
	}
}

// RunSingle executes a single probe run.
func (r *Runtime) RunSingle(ctx context.Context) (errReturn error) {
	metrics.MetricsInstance().IncStartedRuns(r.Config.DataPlaneRegion)
	metrics.MetricsInstance().SetLastStartedTimestamp(r.Config.DataPlaneRegion)

	probeRunCtx, cancel := context.WithTimeout(ctx, r.Config.ProbeRunTimeout)
	defer cancel()
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), r.Config.ProbeCleanUpTimeout)
		defer cancel()

		if err := r.probe.CleanUp(cleanupCtx); err != nil {
			// If clean up failed AND the original probe run failed, wrap the
			// original error and return it in `SingleRun`.
			// If ONLY the clean up failed, the context error is wrapped and
			// returned in `SingleRun`.
			if errReturn != nil {
				errReturn = errors.Wrapf(errReturn, "%s: %s", errCleanupFailed, err)
			} else {
				errReturn = errors.Wrap(err, errCleanupFailed.Error())
			}
		}
	}()

	if err := r.probe.Execute(probeRunCtx); err != nil {
		metrics.MetricsInstance().IncFailedRuns(r.Config.DataPlaneRegion)
		metrics.MetricsInstance().SetLastFailureTimestamp(r.Config.DataPlaneRegion)
		glog.Error("probe run failed: ", err)
		return errors.Wrap(err, "probe run failed")
	}
	metrics.MetricsInstance().IncSucceededRuns(r.Config.DataPlaneRegion)
	metrics.MetricsInstance().SetLastSuccessTimestamp(r.Config.DataPlaneRegion)
	return nil
}
