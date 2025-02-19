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
	"github.com/stackrox/acs-fleet-manager/probe/pkg/central"
	"github.com/stackrox/acs-fleet-manager/probe/pkg/metrics"
	"github.com/stackrox/acs-fleet-manager/probe/pkg/probe"
)

// Runtime orchestrates probe runs against fleet manager.
type Runtime struct {
	config  config.Config
	service central.Service
}

// New creates a new runtime.
func New(config config.Config, service central.Service) *Runtime {
	return &Runtime{
		config:  config,
		service: service,
	}
}

// RunLoop a continuous loop of probe runs.
func (r *Runtime) RunLoop(ctx context.Context) error {
	ticker := time.NewTicker(r.config.ProbeRunWaitPeriod)
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
func (r *Runtime) RunSingle(ctx context.Context) error {
	probeRunCtx, cancel := context.WithTimeout(ctx, r.config.ProbeRunTimeout)
	defer cancel()
	specs, err := r.service.ListSpecs(ctx)
	if err != nil {
		return fmt.Errorf("listing specs: %w", err)
	}
	errCh := make(chan error, len(specs))
	var wg sync.WaitGroup
	for _, spec := range specs {
		wg.Add(1)
		go func(spec central.Spec) {
			defer wg.Done()
			errCh <- r.runWithSpec(probeRunCtx, spec)
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

func (r *Runtime) runWithSpec(ctx context.Context, spec central.Spec) error {
	metrics.MetricsInstance().IncStartedRuns(spec.Region)
	metrics.MetricsInstance().SetLastStartedTimestamp(spec.Region)

	probeInstance := probe.New(r.config, r.service, spec)
	if err := probeInstance.Execute(ctx); err != nil {
		metrics.MetricsInstance().IncFailedRuns(spec.Region)
		metrics.MetricsInstance().SetLastFailureTimestamp(spec.Region)
		glog.Error("probe run failed: ", err)
		return fmt.Errorf("probe run failed: %w", err)
	}
	metrics.MetricsInstance().IncSucceededRuns(spec.Region)
	metrics.MetricsInstance().SetLastSuccessTimestamp(spec.Region)
	return nil
}
