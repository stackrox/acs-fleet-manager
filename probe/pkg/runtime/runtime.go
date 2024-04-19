// Package runtime ...
package runtime

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/stackrox/acs-fleet-manager/probe/config"
	"github.com/stackrox/acs-fleet-manager/probe/pkg/metrics"
	"github.com/stackrox/acs-fleet-manager/probe/pkg/probe"
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
			return fmt.Errorf("probe context invalid: %w", ctx.Err())
		case <-ticker.C:
			if err := r.RunSingle(ctx); err != nil {
				glog.Error(err)
			}
		}
	}
}

// RunSingle executes a single probe run.
func (r *Runtime) RunSingle(ctx context.Context) (errReturn error) {
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), r.Config.ProbeCleanUpTimeout)
		defer cancel()

		if err := r.probe.CleanUp(cleanupCtx); err != nil {
			// If clean up failed AND the original probe run failed, wrap the
			// original error and return it in `SingleRun`.
			// If ONLY the clean up failed, the context error is wrapped and
			// returned in `SingleRun`.
			if errReturn != nil {
				errReturn = fmt.Errorf("cleanup failed: %w: %w", err, errReturn)
			} else {
				errReturn = fmt.Errorf("cleanup failed: %w", err)
			}
		}
	}()
	var wg sync.WaitGroup
	errCh := make(chan error, len(r.Config.CentralSpecs))

	for _, spec := range r.Config.CentralSpecs {
		wg.Add(1)
		go func(spec config.CentralSpec) {
			defer wg.Done()
			errCh <- r.runWithSpec(ctx, spec)
		}(spec)
	}

	wg.Wait()
	close(errCh)

	var result error
	for err := range errCh {
		result = errors.Join(result, err)
	}
	return result
}

func (r *Runtime) runWithSpec(ctx context.Context, spec config.CentralSpec) error {
	metrics.MetricsInstance().IncStartedRuns(spec.Region)
	metrics.MetricsInstance().SetLastStartedTimestamp(spec.Region)

	probeRunCtx, cancel := context.WithTimeout(ctx, r.Config.ProbeRunTimeout)
	defer cancel()

	if err := r.probe.Execute(probeRunCtx, spec); err != nil {
		metrics.MetricsInstance().IncFailedRuns(spec.Region)
		metrics.MetricsInstance().SetLastFailureTimestamp(spec.Region)
		glog.Error("probe run failed: ", err)
		return fmt.Errorf("probe run failed: %w", err)
	}
	metrics.MetricsInstance().IncSucceededRuns(spec.Region)
	metrics.MetricsInstance().SetLastSuccessTimestamp(spec.Region)
	return nil
}
